//go:build windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"
)

func TestDrawItemStructUsesNativeArchitectureLayout(t *testing.T) {
	got := unsafe.Sizeof(drawItemStruct{})
	var want uintptr
	switch runtime.GOARCH {
	case "386":
		want = 48
	case "amd64":
		want = 64
	default:
		t.Skipf("layout not asserted for GOARCH=%s", runtime.GOARCH)
	}
	if got != want {
		t.Fatalf("DRAWITEMSTRUCT layout invalid on %s: got %d bytes, want %d; manual padding can break WM_DRAWITEM", runtime.GOARCH, got, want)
	}
}

func TestParseDatabaseHeader(t *testing.T) {
	output := "Page size\t\t4096\r\nODS version\t\t13.1\r\nDatabase dialect\t1\r\n"
	h := parseDatabaseHeader(output)
	if h.PageSize != "4096" || h.ODS != "13.1" || h.Dialect != "1" {
		t.Fatalf("unexpected header: %#v", h)
	}
}

func TestSummaryOutputFilter(t *testing.T) {
	if shouldSurfaceSummaryOutput("restoring privilege for user PUBLIC") {
		t.Fatal("normal verbose line should stay hidden in summary mode")
	}
	if !shouldSurfaceSummaryOutput("ERROR: database file appears corrupt") {
		t.Fatal("errors must remain visible in summary mode")
	}
}

func TestEffectiveCommandArgsRemovesGbakVerbose(t *testing.T) {
	old := detailedLog
	detailedLog = false
	defer func() { detailedLog = old }()
	args := effectiveCommandArgs("gbak.exe", []string{"-b", "-v", "source", "backup"})
	for _, arg := range args {
		if arg == "-v" {
			t.Fatal("-v must be removed from gbak in summary mode")
		}
	}
}

func TestSuccessSummaryIncludesRestoreMetadata(t *testing.T) {
	oldHeader, oldBackup, oldReport := runResultSnapshot()
	defer func() {
		setLastHeader(oldHeader)
		setLastBackup(oldBackup)
		setLastReport(oldReport)
	}()
	setLastHeader(databaseHeader{ODS: "13.1", Dialect: "1", PageSize: "4096"})
	setLastBackup(`C:\Backup\DBCOM_20260810_101539.BAK`)
	setLastReport(`C:\Backup\RECOVERY_DBCOM\RELATORIO_RECUPERACAO.txt`)
	c := workflowConfig{
		Routine:       routines[0],
		TargetVersion: "5.0",
		Dest:          `C:\Dados\DBCOM50.RED`,
	}
	text, compact := successSummary(c)
	for _, want := range []string{"Rotina concluída com sucesso", "Firebird de destino: 5.0", "ODS: 13.1", "Dialeto: 1", `C:\Dados\DBCOM50.RED`, "RELATORIO_RECUPERACAO.txt"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q: %s", want, text)
		}
	}
	if !strings.Contains(compact, "ODS 13.1") || !strings.Contains(compact, "Dialeto 1") {
		t.Fatalf("unexpected compact summary: %s", compact)
	}
}

func TestPasswordEditStyleToggle(t *testing.T) {
	base := uint32(WS_CHILD | WS_VISIBLE | ES_AUTOHSCROLL)
	hidden := passwordEditStyle(base, false)
	if hidden&ES_PASSWORD == 0 {
		t.Fatal("hidden password style must include ES_PASSWORD")
	}
	visible := passwordEditStyle(hidden, true)
	if visible&ES_PASSWORD != 0 {
		t.Fatal("visible password style must remove ES_PASSWORD")
	}
}

func TestSupportedFirebirdVersions(t *testing.T) {
	want := []string{"1.0", "1.5", "2.0", "2.1", "2.5", "3.0", "4.0", "5.0"}
	if len(versions) != len(want) {
		t.Fatalf("unexpected version count: got %d want %d", len(versions), len(want))
	}
	for i := range want {
		if versions[i] != want[i] {
			t.Fatalf("version[%d] = %q, want %q", i, versions[i], want[i])
		}
	}
}

func TestLegacyVersionsUseBinLayout(t *testing.T) {
	for _, version := range []string{"1.0", "1.5", "2.0", "2.1", "2.5"} {
		if !versionUsesBin(version) {
			t.Fatalf("Firebird %s should prefer bin layout", version)
		}
	}
	for _, version := range []string{"3.0", "4.0", "5.0"} {
		if versionUsesBin(version) {
			t.Fatalf("Firebird %s should prefer root layout", version)
		}
	}
}

func TestBackwardMigrationDetection(t *testing.T) {
	if !isBackwardMigration("5.0", "2.5") {
		t.Fatal("5.0 -> 2.5 must be detected as backward migration")
	}
	if isBackwardMigration("2.5", "5.0") {
		t.Fatal("2.5 -> 5.0 must not be detected as backward migration")
	}
	if isBackwardMigration("4.0", "4.0") {
		t.Fatal("same-version restore must not be detected as backward migration")
	}
}

func TestDestinationSuffixesCoverAllVersions(t *testing.T) {
	cases := map[string]string{
		"1.0": "10.RED",
		"1.5": "15.RED",
		"2.0": "20.RED",
		"2.1": "21.RED",
		"2.5": "25.RED",
		"3.0": "30.RED",
		"4.0": "40.RED",
		"5.0": "50.RED",
	}
	for version, want := range cases {
		if got := suffixForVersion(version); got != want {
			t.Fatalf("suffixForVersion(%s) = %q, want %q", version, got, want)
		}
	}
}

func TestOnlyThreeRoutinesRemain(t *testing.T) {
	want := []string{"Manutenção / conversão", "Recuperar banco corrompido", "Somente restaurar"}
	if len(routines) != len(want) {
		t.Fatalf("unexpected routine count: got %d want %d", len(routines), len(want))
	}
	for i := range want {
		if routines[i].Name != want[i] {
			t.Fatalf("routine[%d] = %q, want %q", i, routines[i].Name, want[i])
		}
	}
}

func TestLegacyRoutinePreferenceAliases(t *testing.T) {
	items := routineNames(routines)
	for legacy, want := range map[string]int{
		"Corrigir banco":            0,
		"Corrigir banco (avançado)": 1,
	} {
		if got := preferenceRoutineIndex(items, legacy); got != want {
			t.Fatalf("legacy routine %q = %d, want %d", legacy, got, want)
		}
	}
}

func TestRoutineNamesMatchesRoutineDefinitions(t *testing.T) {
	names := routineNames(routines)
	if len(names) != len(routines) {
		t.Fatalf("routine names count mismatch: got %d want %d", len(names), len(routines))
	}
	for i, routine := range routines {
		if names[i] != routine.Name {
			t.Fatalf("routineNames[%d] = %q, want %q", i, names[i], routine.Name)
		}
	}
}

func TestParseRecoveryStats(t *testing.T) {
	output := "Number of record level errors 2\r\nNumber of index page errors 14\r\nNumber of transaction page errors 1\r\nNumber of BLOB errors 3\r\nNumber of database page errors 7\r\n"
	stats := parseRecoveryStats(output)
	if stats.RecordErrors != 2 || stats.IndexErrors != 14 || stats.TransactionErrors != 1 || stats.BlobErrors != 3 || stats.DatabaseErrors != 7 {
		t.Fatalf("unexpected recovery stats: %#v", stats)
	}
	if !stats.HasCorruption() || stats.Severity() != "ALTA" {
		t.Fatalf("expected high-severity corruption: %#v", stats)
	}
}

func TestIndexOnlyRecoveryStats(t *testing.T) {
	stats := parseRecoveryStats("Number of index page errors 8\r\n")
	if !stats.IndexOnly() {
		t.Fatalf("expected index-only classification: %#v", stats)
	}
	if stats.Severity() != "BAIXA (índices)" {
		t.Fatalf("unexpected severity: %s", stats.Severity())
	}
}

func TestCountLimboTransactions(t *testing.T) {
	output := "Transaction 67 is in limbo.\r\nTransaction 91 is in limbo.\r\n"
	if got := countLimboTransactions(output); got != 2 {
		t.Fatalf("limbo count = %d, want 2", got)
	}
}

func TestAdvancedBackupArgsUsesSkipDataOnFirebird3Plus(t *testing.T) {
	c := workflowConfig{SourceVersion: "3.0", User: "SYSDBA", Password: "secret", SkipData: "(BAD_TABLE|OLD_DATA)"}
	args := advancedBackupArgs(c, "db", "backup", true, true)
	joined := strings.Join(args, " ")
	for _, want := range []string{"-ignore", "-limbo", "-skip_data", "(BAD_TABLE|OLD_DATA)"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("advanced args missing %q: %v", want, args)
		}
	}
}

func TestAdvancedBackupArgsDoesNotUseSkipDataBeforeFirebird3(t *testing.T) {
	c := workflowConfig{SourceVersion: "2.5", User: "SYSDBA", Password: "secret", SkipData: "BAD_TABLE"}
	args := advancedBackupArgs(c, "db", "backup", true, false)
	if strings.Contains(strings.Join(args, " "), "-skip_data") {
		t.Fatalf("-skip_data must not be used before Firebird 3.0: %v", args)
	}
}

func TestRestoreTestArgsOneAtATime(t *testing.T) {
	args := restoreTestArgs("5.0", "3055", "SYSDBA", "secret", "backup", "dest", true)
	if !strings.Contains(strings.Join(args, " "), "-one_at_a_time") {
		t.Fatalf("restore test args must include -one_at_a_time: %v", args)
	}
}

func TestToolOutputOperationalFailureForLegacyExitCode(t *testing.T) {
	if !toolOutputHasOperationalFailure("failed to attach to database") {
		t.Fatal("legacy textual failure must be detected even if process exit code is zero")
	}
	if toolOutputHasOperationalFailure("Number of index page errors 4") {
		t.Fatal("corruption summary is diagnostic, not an operational invocation failure")
	}
}

func TestNormalizeSkipData(t *testing.T) {
	for input, want := range map[string]string{
		` "(BAD|OLD)" `: "(BAD|OLD)",
		`'(A|B)'`:       "(A|B)",
		`TABLE_X`:       "TABLE_X",
	} {
		if got := normalizeSkipData(input); got != want {
			t.Fatalf("normalizeSkipData(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFailureSummaryIncludesRecoveryReport(t *testing.T) {
	oldHeader, oldBackup, oldReport := runResultSnapshot()
	defer func() {
		setLastHeader(oldHeader)
		setLastBackup(oldBackup)
		setLastReport(oldReport)
	}()
	setLastReport(`C:\Recovery\RELATORIO_RECUPERACAO.txt`)
	text := failureSummary(errors.New("restore de teste falhou"))
	if !strings.Contains(text, "RELATORIO_RECUPERACAO.txt") {
		t.Fatalf("failure summary missing recovery report: %s", text)
	}
}

func TestOutputReportsUnknownSwitch(t *testing.T) {
	for _, output := range []string{"unknown switch -no_update", "invalid option: -no_update"} {
		if !outputReportsUnknownSwitch(output) {
			t.Fatalf("expected unsupported switch detection for %q", output)
		}
	}
}

func TestMergeRecoveryStatsKeepsWorstCounts(t *testing.T) {
	a := recoveryStats{IndexErrors: 3, TransactionErrors: 1}
	b := recoveryStats{IndexErrors: 2, BlobErrors: 4, GenericCorruption: true}
	got := mergeRecoveryStats(a, b)
	if got.IndexErrors != 3 || got.TransactionErrors != 1 || got.BlobErrors != 4 || !got.GenericCorruption {
		t.Fatalf("unexpected merged recovery stats: %#v", got)
	}
}

func TestProjectCreditAndVersion(t *testing.T) {
	if appVersion != "1.2.31" {
		t.Fatalf("unexpected app version: %s", appVersion)
	}
	if appCredit != "Feito por Marcos De Mori Laiola" {
		t.Fatalf("unexpected project credit: %s", appCredit)
	}
}

func TestUILayoutReadabilityMinimums(t *testing.T) {
	if uiFilesLabelWidth < 145 {
		t.Fatalf("file labels are too narrow: %d", uiFilesLabelWidth)
	}
	if uiFirebirdLabelWidth < 155 {
		t.Fatalf("Firebird labels are too narrow: %d", uiFirebirdLabelWidth)
	}
	if uiProgressTextWidth < 700 {
		t.Fatalf("progress text area is too narrow: %d", uiProgressTextWidth)
	}
	if uiProgressBarWidth < 700 {
		t.Fatalf("progress bar is too narrow: %d", uiProgressBarWidth)
	}
	if uiElapsedTextWidth < 500 {
		t.Fatalf("elapsed time area is too narrow: %d", uiElapsedTextWidth)
	}
	if uiSkipEditWidth < 340 {
		t.Fatalf("skip_data edit is too narrow: %d", uiSkipEditWidth)
	}
	if uiSkipEditX+uiSkipEditWidth+10 > uiOpenReportX {
		t.Fatalf("skip_data edit overlaps result shortcuts: editEnd=%d reportX=%d", uiSkipEditX+uiSkipEditWidth, uiOpenReportX)
	}
	if uiOpenReportWidth < 145 {
		t.Fatalf("open report button is too narrow: %d", uiOpenReportWidth)
	}
	if uiSectionBorderLift < 6 || uiSectionBorderLift > 10 {
		t.Fatalf("section border lift must keep the top line above headers without changing layout: %d", uiSectionBorderLift)
	}
	if uiLocateDatabaseWidth < 160 {
		t.Fatalf("locate database button is too narrow: %d", uiLocateDatabaseWidth)
	}
	if uiLocateDatabaseX+uiLocateDatabaseWidth > 884 {
		t.Fatalf("result shortcuts exceed execution area: x=%d width=%d", uiLocateDatabaseX, uiLocateDatabaseWidth)
	}
}

func TestResultShortcutsGainAccentWhenAvailable(t *testing.T) {
	for _, id := range []int{idOpenReport, idOpenDatabase} {
		visual := buttonVisuals[id]
		if !visual.Accent {
			t.Fatalf("result shortcut %d must use accent visual when enabled", id)
		}
	}
}

func TestSectionHeadersDoNotHideGroupboxTopBorder(t *testing.T) {
	limits := map[int]int32{
		idSectionRoutine:  170,
		idSectionFiles:    190,
		idSectionFirebird: 250,
		idSectionExecute:  125,
		idSectionLog:      160,
	}
	for id, maxWidth := range limits {
		width := sectionHeaderWidth(id)
		if width <= 0 || width > maxWidth {
			t.Fatalf("section header %d hides too much of the groupbox border: width=%d max=%d", id, width, maxWidth)
		}
	}
}

func TestHelpLayoutFitsCompactWindow(t *testing.T) {
	if uiHelpWindowWidth > 900 || uiHelpWindowHeight > 720 {
		t.Fatalf("help window is no longer compact: %dx%d", uiHelpWindowWidth, uiHelpWindowHeight)
	}
	if uiHelpNavWidth < 160 || uiHelpContentWidth < 540 || uiHelpContentHeight < 360 {
		t.Fatalf("help layout lost readable space: nav=%d content=%dx%d", uiHelpNavWidth, uiHelpContentWidth, uiHelpContentHeight)
	}
	if uiHelpBottomY+40 >= uiHelpWindowHeight {
		t.Fatalf("help bottom actions can be clipped: bottomY=%d windowHeight=%d", uiHelpBottomY, uiHelpWindowHeight)
	}
}

func TestHelpSectionsProvideScannableNavigation(t *testing.T) {
	sections := helpSections()
	if len(sections) != 5 {
		t.Fatalf("unexpected help section count: %d", len(sections))
	}
	for _, want := range []string{"Comece por aqui", "Rotinas", "Recuperação avançada", "Resultados e saúde", "Erros e segurança"} {
		found := false
		for _, section := range sections {
			if section.Title == want && strings.TrimSpace(section.Content) != "" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("help section missing or empty: %s", want)
		}
	}
}

func TestDetailedLogButtonUsesClearLabels(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{`setText(idDetailedLog, "Ver detalhes")`, `setText(idDetailedLog, "Resumir log")`, `createIconButton(idDetailedLog, "Ver detalhes"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("detailed log control missing clear label %q", want)
		}
	}
}

func TestFormatElapsedDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0 min 00 s"},
		{59 * time.Second, "0 min 59 s"},
		{65 * time.Second, "1 min 05 s"},
		{2*time.Minute + 7*time.Second, "2 min 07 s"},
	}
	for _, tc := range cases {
		if got := formatElapsedDuration(tc.d); got != tc.want {
			t.Fatalf("formatElapsedDuration(%s) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestExecutionTimerIsWiredIntoWorkflowAndUI(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, want := range []string{
		`startRunTimer()`,
		`finishRunTimer()`,
		`"Tempo decorrido:"`,
		`"Tempo total de execução: "`,
		`"Tempo total: "+elapsedTextSnapshot()`,
		`idElapsed`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("execution timer wiring missing %q", want)
		}
	}
	if strings.Count(text, "finishRunTimer()") < 4 {
		t.Fatalf("timer must be finalized in cancel, error and success paths")
	}
}

func TestTerminalProgressHelperIsUsedForAllFinalStates(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, expected := range []string{
		`setTerminalProgress(PBST_PAUSED, "Interrompido pelo usuário")`,
		`setTerminalProgress(PBST_PAUSED, "Aguardando decisão manual")`,
		`setTerminalProgress(PBST_PAUSED, "Recuperação parcial disponível")`,
		`setTerminalProgress(PBST_ERROR, "Encerrado com erro")`,
		`setTerminalProgress(PBST_ERROR, "Não apto para produção")`,
		`setTerminalProgress(PBST_PAUSED, "Recuperação parcial concluída")`,
		`setTerminalProgress(PBST_NORMAL, "Concluído com sucesso")`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("terminal progress branch missing: %s", expected)
		}
	}
	if !strings.Contains(text, `PBM_SETPOS, 100`) || !strings.Contains(text, `PBM_SETPOS, 99`) {
		t.Fatal("terminal progress must force the native Windows progress bar to its final visual position")
	}
}

func TestGbakLooksLikeIndexCorruption(t *testing.T) {
	if !gbakLooksLikeIndexCorruption("page 213030 is of wrong type (expected index B-tree, found blob)") {
		t.Fatal("expected index-page corruption detection")
	}
	if gbakLooksLikeIndexCorruption("record 42 is corrupt") {
		t.Fatal("record corruption must not be classified as index-page corruption")
	}
}

func TestIntegerPrimaryKey(t *testing.T) {
	if !integerPrimaryKey(relationMetadata{PrimaryKey: "COD_IT", PrimaryType: 8, PrimaryScale: 0}) {
		t.Fatal("INTEGER primary key should be eligible for range rescue")
	}
	if integerPrimaryKey(relationMetadata{PrimaryKey: "COD_IT", PrimaryType: 16, PrimaryScale: -2}) {
		t.Fatal("scaled numeric key should not be treated as integer range")
	}
	if integerPrimaryKey(relationMetadata{PrimaryKey: "COD", PrimaryType: 37, PrimaryScale: 0}) {
		t.Fatal("VARCHAR key should not be eligible for integer range rescue")
	}
}

func TestCopyRelationBlockUsesNaturalReadAndExternalInsert(t *testing.T) {
	meta := relationMetadata{
		Name:    "DOCIT",
		Columns: []relationColumn{{Name: "COD_IT"}, {Name: "DESCRICAO"}},
	}
	c := workflowConfig{TargetVersion: "5.0", TargetPort: "3055", User: "SYSDBA", Password: "secret"}
	script, err := copyRelationBlock(meta, c, `C:\Temp\stage.fdb`, "", true)
	if err != nil {
		t.Fatalf("copyRelationBlock failed: %v", err)
	}
	for _, want := range []string{
		`PLAN (T NATURAL)`,
		`INSERT INTO "DOCIT" ("COD_IT", "DESCRICAO") VALUES (?, ?)`,
		`ON EXTERNAL DATA SOURCE`,
		`FTB_ROWS=`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("generated rescue script missing %q:\n%s", want, script)
		}
	}
}

func TestCopyRelationBlockRejectsArrays(t *testing.T) {
	meta := relationMetadata{Name: "T", Columns: []relationColumn{{Name: "A", HasArray: true}}}
	if _, err := copyRelationBlock(meta, workflowConfig{}, `C:\x.fdb`, "", true); err == nil {
		t.Fatal("array column should block automatic row rescue")
	}
}

func TestParseRecoveredRows(t *testing.T) {
	if got := parseRecoveredRows("FTB_RESULT  FTB_ROWS=202047"); got != 202047 {
		t.Fatalf("parseRecoveredRows = %d, want 202047", got)
	}
}

func TestISQLOutputHasFailure(t *testing.T) {
	for _, text := range []string{
		"Statement failed, SQLSTATE = 08000",
		"Dynamic SQL Error SQL error code = -104",
		"database file appears corrupt",
	} {
		if !isqlOutputHasFailure(text) {
			t.Fatalf("expected isql failure detection for %q", text)
		}
	}
	if isqlOutputHasFailure("FTB_ROWS=100\nCommit complete.") {
		t.Fatal("successful rescue output must not be treated as failure")
	}
}

func TestSuspectedRelationFromGbakOutput(t *testing.T) {
	output := "gbak: writing data for table CLIENTES\r\ngbak: writing data for table MOVIMENTO_ESTOQUE\r\ngbak: ERROR: page is of wrong type\r\n"
	if got := suspectedRelationFromGbak(output); got != "MOVIMENTO_ESTOQUE" {
		t.Fatalf("suspected relation = %q, want MOVIMENTO_ESTOQUE", got)
	}
}

func TestSuspectedRelationFromExplicitError(t *testing.T) {
	output := "gbak: ERROR: relation \"PEDIDOS ITENS\" is corrupt\r\n"
	if got := suspectedRelationFromGbak(output); got != "PEDIDOS ITENS" {
		t.Fatalf("suspected relation = %q, want PEDIDOS ITENS", got)
	}
}

func TestSafeRelationForSkipRejectsSystemTables(t *testing.T) {
	for _, name := range []string{"RDB$RELATIONS", "MON$ATTACHMENTS", "SEC$USERS"} {
		if safeRelationForSkip(name) {
			t.Fatalf("system relation must not be auto-skipped: %s", name)
		}
	}
	if !safeRelationForSkip("MOVIMENTO_ESTOQUE") {
		t.Fatal("user relation should be eligible for explicit skip confirmation")
	}
}

func TestSQLRegexLiteralEscapesWildcards(t *testing.T) {
	got := sqlRegexLiteral("MOVIMENTO_ESTOQUE%OLD")
	if got != "MOVIMENTO[_]ESTOQUE[%]OLD" {
		t.Fatalf("unexpected SQL regex literal: %q", got)
	}
}

func TestRelevantGbakFailureLinesKeepsErrorContext(t *testing.T) {
	output := strings.Join([]string{
		"gbak: writing data for table A",
		"gbak: 100 records written",
		"gbak: writing data for table MOVIMENTO",
		"gbak: ERROR: database file appears corrupt",
		"gbak: ERROR: page is of wrong type",
	}, "\n")
	got := relevantGbakFailureLines(output, 10)
	for _, want := range []string{"MOVIMENTO", "appears corrupt", "wrong type"} {
		if !strings.Contains(got, want) {
			t.Fatalf("failure details missing %q: %s", want, got)
		}
	}
}

func TestMetadataBackupArgs(t *testing.T) {
	c := workflowConfig{User: "SYSDBA", Password: "secret"}
	args := metadataBackupArgs(c, "db", "meta.bak")
	joined := strings.Join(args, " ")
	for _, want := range []string{"-b", "-v", "-m", "db", "meta.bak"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("metadata backup args missing %q: %v", want, args)
		}
	}
}

func TestRecoveryCommandArgsPreservesGbakVerbose(t *testing.T) {
	args := recoveryCommandArgs("gbak.exe", []string{"-b", "-v", "db", "backup"})
	if !strings.Contains(strings.Join(args, " "), "-v") {
		t.Fatalf("advanced recovery must preserve gbak verbose internally: %v", args)
	}
}

func TestRecoveryStatsComparison(t *testing.T) {
	before := recoveryStats{IndexErrors: 6, DatabaseErrors: 1}
	after := recoveryStats{DatabaseErrors: 1}
	got := recoveryStatsComparison(before, after)
	if !strings.Contains(got, "índices=6") || !strings.Contains(got, "Depois:") || !strings.Contains(got, "índices=0") {
		t.Fatalf("unexpected comparison: %s", got)
	}
}

func TestSuspectedRelationIgnoresGenericTableWords(t *testing.T) {
	output := "gbak: ERROR: table is corrupt\r\n"
	if got := suspectedRelationFromGbak(output); got != "" {
		t.Fatalf("generic grammar must not be treated as relation name: %q", got)
	}
}

func TestHelpTextCoversCoreWorkflows(t *testing.T) {
	text := helpText()
	for _, want := range []string{
		"Manutenção / conversão",
		"Recuperar banco corrompido",
		"Somente restaurar",
		"PULAR TABELAS (REGEX)",
		"RECUPERAÇÃO PARCIAL",
		"RELATORIO_RECUPERACAO.txt",
		"LOG_TECNICO_RECUPERACAO.txt",
		"Feito por Marcos De Mori Laiola",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help text missing %q", want)
		}
	}
}

func TestSalvageEngineAlwaysUsesSourceVersion(t *testing.T) {
	c := workflowConfig{
		SourceVersion: "5.0",
		FbSource:      `C:\Firebird\5`,
		SourcePort:    "3055",
		TargetVersion: "2.5",
		FbTarget:      `C:\Firebird\2_5`,
		TargetPort:    "3050",
	}
	version, root, port := salvageEngineConfig(c)
	if version != "5.0" || root != `C:\Firebird\5` || port != "3055" {
		t.Fatalf("salvage engine must follow source engine, got %s | %s | %s", version, root, port)
	}
}

func TestParseRestoreDuplicateConflicts(t *testing.T) {
	output := `gbak:cannot commit index RDB$PRIMARY12
` +
		`gbak: ERROR:violation of PRIMARY or UNIQUE KEY constraint "DOCIT_PRKEY" on table "DOCIT"
` +
		`gbak: ERROR:    Problematic key value is ("COD_DOC" = 31432, "COD_DOCIT" = 72)
`
	conflicts := parseRestoreDuplicateConflicts(output)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %#v", len(conflicts), conflicts)
	}
	got := conflicts[0]
	if got.Constraint != "DOCIT_PRKEY" || got.Relation != "DOCIT" {
		t.Fatalf("unexpected conflict: %#v", got)
	}
	if len(got.KeyParts) != 2 || got.KeyParts[0].Column != "COD_DOC" || got.KeyParts[0].RawValue != "31432" || got.KeyParts[1].Column != "COD_DOCIT" || got.KeyParts[1].RawValue != "72" {
		t.Fatalf("unexpected key parts: %#v", got.KeyParts)
	}
	if label := duplicateConflictKeyLabel(got); label != "COD_DOC=31432 / COD_DOCIT=72" {
		t.Fatalf("unexpected key label: %s", label)
	}
}

func TestDuplicateKeyPredicateCompositeKey(t *testing.T) {
	conflict := restoreDuplicateConflict{KeyParts: []duplicateKeyPart{
		{Column: "COD_DOC", RawValue: "31432"},
		{Column: "COD_DOCIT", RawValue: "72"},
	}}
	where, err := duplicateKeyPredicate(conflict, "T")
	if err != nil {
		t.Fatalf("duplicateKeyPredicate failed: %v", err)
	}
	for _, want := range []string{`T."COD_DOC" = 31432`, `T."COD_DOCIT" = 72`} {
		if !strings.Contains(where, want) {
			t.Fatalf("predicate missing %q: %s", want, where)
		}
	}
}

func TestDuplicateCleanupScriptKeepsOnePhysicalOccurrence(t *testing.T) {
	conflict := restoreDuplicateConflict{
		Relation: "DOCIT",
		KeyParts: []duplicateKeyPart{{Column: "COD_DOC", RawValue: "31432"}, {Column: "COD_DOCIT", RawValue: "72"}},
	}
	script, err := duplicateCleanupScript(conflict)
	if err != nil {
		t.Fatalf("duplicateCleanupScript failed: %v", err)
	}
	for _, want := range []string{`RDB$DB_KEY`, `PLAN (T NATURAL)`, `IF (N = 1)`, `K <> KEEP_K`, `DELETE FROM "DOCIT"`} {
		if !strings.Contains(script, want) {
			t.Fatalf("cleanup script missing %q:\n%s", want, script)
		}
	}
}

func TestDuplicateCleanupCanKeepSecondOccurrence(t *testing.T) {
	conflict := restoreDuplicateConflict{
		Relation: "DOCIT",
		KeyParts: []duplicateKeyPart{{Column: "COD_DOC", RawValue: "31432"}, {Column: "COD_DOCIT", RawValue: "72"}},
	}
	script, err := duplicateCleanupKeepOccurrenceScript(conflict, 2)
	if err != nil {
		t.Fatalf("duplicateCleanupKeepOccurrenceScript failed: %v", err)
	}
	if !strings.Contains(script, `IF (N = 2) THEN KEEP_K = K`) {
		t.Fatalf("cleanup script does not preserve second physical record:\n%s", script)
	}
}

func TestDuplicateDetailUsesSHA256ForBlobsOnFirebird5(t *testing.T) {
	meta := relationMetadata{Name: "DOCIT", Columns: []relationColumn{{Name: "DTINS_DOCIT", FieldType: 35}, {Name: "FOTO_DOCIT", FieldType: 261}}}
	conflict := restoreDuplicateConflict{Relation: "DOCIT", KeyParts: []duplicateKeyPart{{Column: "COD_DOC", RawValue: "31432"}}}
	script, err := duplicateOccurrenceDetailScript(meta, conflict, 1, meta.Columns[:1], "5.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"CRYPT_HASH", "SHA256", "HEX_ENCODE", "FTB_BLOB|FOTO_DOCIT|", "PLAN (T NATURAL)"} {
		if !strings.Contains(script, want) {
			t.Fatalf("detail script missing %q:\n%s", want, script)
		}
	}
}

func TestDuplicateOccurrenceFormattingHighlightsDifferences(t *testing.T) {
	details := duplicateOccurrenceDetails{Ordinal: 1, Values: map[string]string{"COD_DOC": "31432", "DTINS_DOCIT": "2026-08-07 16:44:27"}}
	cols := []relationColumn{{Name: "COD_DOC", FieldType: 8}, {Name: "DTINS_DOCIT", FieldType: 35}}
	conflict := restoreDuplicateConflict{DifferingColumns: []string{"DTINS_DOCIT"}}
	text := formatDuplicateOccurrence(details, cols, conflict)
	if !strings.Contains(text, "[DIFERENTE] DTINS_DOCIT") {
		t.Fatalf("difference marker missing:\n%s", text)
	}
	if strings.Contains(text, "[DIFERENTE] COD_DOC") {
		t.Fatalf("unchanged key was incorrectly highlighted:\n%s", text)
	}
}

func TestBlobColumnMakesDuplicateComparisonInconclusive(t *testing.T) {
	if columnIsConflictComparable(relationColumn{Name: "ANEXO", FieldType: 261}) {
		t.Fatal("BLOB columns must not be used for automatic duplicate equivalence")
	}
	if !columnIsConflictComparable(relationColumn{Name: "COD_DOC", FieldType: 8}) {
		t.Fatal("regular integer column should be comparable")
	}
}

func TestRestoreInactiveArgs(t *testing.T) {
	args := restoreInactiveArgs("5.0", "3055", "SYSDBA", "secret", "backup", "dest")
	joined := strings.Join(args, " ")
	for _, want := range []string{"-c", "-v", "-inactive", "service_mgr"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("inactive restore args missing %q: %v", want, args)
		}
	}
}

func TestHelpTextCoversDuplicateKeyRecovery(t *testing.T) {
	text := helpText()
	for _, want := range []string{"PRIMARY KEY", "UNIQUE", "-inactive", "CONFLITOS_DUPLICIDADE.txt", "primeira ocorrência", "REGISTRO 1", "REGISTRO 2", "SHA-256"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help text missing duplicate-recovery concept %q", want)
		}
	}
}

func TestWindowsDistributionMetadataConfig(t *testing.T) {
	data, err := os.ReadFile("winres/winres.json")
	if err != nil {
		t.Fatalf("winres metadata missing: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		`"CompanyName": "Marcos De Mori Laiola"`,
		`"ProductName": "Firebird Toolbox"`,
		`"FileDescription": "Firebird Toolbox - manutenção, conversão e recuperação Firebird"`,
		`"FileVersion": "1.2.31.0"`,
		`"ProductVersion": "1.2.31"`,
		`"execution-level": "as invoker"`,
		`"use-common-controls-v6": true`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("winres metadata missing %q", want)
		}
	}
}

func TestWindowsBuildScriptsPassLdflagsAsSeparateArgument(t *testing.T) {
	paths := []string{
		".github/workflows/release.yml",
		".github/workflows/ci.yml",
		"tools/build.ps1",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(data)
		if strings.Contains(text, "-ldflags=$ld") {
			t.Fatalf("%s still uses PowerShell-unsafe -ldflags=$ld syntax", path)
		}
		if !strings.Contains(text, "'-ldflags'") && !strings.Contains(text, `"-ldflags"`) {
			t.Fatalf("%s must pass -ldflags as its own argument", path)
		}
		if !strings.Contains(text, "$ld") {
			t.Fatalf("%s must pass the ldflags value separately", path)
		}
	}
}

func TestWindowsBuildVerifierRequiresIconAndManifest(t *testing.T) {
	data, err := os.ReadFile("tools/verify-windows-build.ps1")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"RT_GROUP_ICON (14)", "RT_MANIFEST (24)", "Diretorio de recursos Windows ausente"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Windows build verifier missing resource gate %q", want)
		}
	}
}

func TestEmbeddedIconsStayInMemory(t *testing.T) {
	data, err := embeddedFiles.ReadFile("assets/flame.ico")
	if err != nil {
		t.Fatalf("embedded flame icon missing: %v", err)
	}
	image, ok := icoImageData(data, 32)
	if !ok || len(image) == 0 {
		t.Fatal("embedded icon could not be parsed for in-memory loading")
	}

	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, forbidden := range []string{"prepareUIAssets", "LR_LOADFROMFILE", `os.TempDir(), "FirebirdToolbox"`, `u16("runas")`, "IsUserAnAdmin"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("distribution hardening regression: main.go still contains %q", forbidden)
		}
	}
	for _, required := range []string{"CreateIconFromResourceEx", "icoImageData", "execution-level"} {
		if required == "execution-level" {
			continue
		}
		if !strings.Contains(text, required) {
			t.Fatalf("distribution hardening missing %q", required)
		}
	}
}

func TestReleaseRequiresMicrosoftArtifactSigning(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"id-token: write",
		"attestations: write",
		"azure/login@v3",
		"azure/artifact-signing-action@v2",
		"timestamp.acs.microsoft.com",
		"actions/attest@v4",
		"Release bloqueada: configure Artifact Signing antes de publicar",
		"-RequireSignature -RequireTimestamp",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release workflow missing mandatory signing control %q", want)
		}
	}
	for _, forbidden := range []string{"WINDOWS_SIGN_CERT_BASE64", "WINDOWS_SIGN_CERT_PASSWORD", "UNSIGNED -", "Release sera publicada sem assinatura"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("release workflow must not permit unsigned publication: found %q", forbidden)
		}
	}
}

func TestLocalBuildMarksUnsignedArtifactsAsDevelopmentOnly(t *testing.T) {
	data, err := os.ReadFile("tools/build.ps1")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"[switch]$AllowUnsignedDev", "Build de distribuicao bloqueado sem assinatura", "_UNSIGNED_DEV", "NAO DISTRIBUIR"} {
		if !strings.Contains(text, want) {
			t.Fatalf("local build hardening missing %q", want)
		}
	}
}

func TestParseDatabaseHeaderIncludesTransactionIndicators(t *testing.T) {
	out := `Database header page information:
ODS version 13.1
Page size 4096
Oldest transaction 100
Oldest active 101
Oldest snapshot 102
Next transaction 120
Database dialect 1
Sweep interval 20000`
	h := parseDatabaseHeader(out)
	if h.ODS != "13.1" || h.Dialect != "1" || h.PageSize != "4096" {
		t.Fatalf("unexpected basic header: %#v", h)
	}
	if h.OldestTransaction != "100" || h.OldestActive != "101" || h.OldestSnapshot != "102" || h.NextTransaction != "120" || h.SweepInterval != "20000" {
		t.Fatalf("transaction indicators missing: %#v", h)
	}
	if gap, ok := headerTransactionGap(h); !ok || gap != 20 {
		t.Fatalf("unexpected transaction gap: %d ok=%v", gap, ok)
	}
}

func TestHealthMetadataScriptCoversCriticalIndexesAndConstraints(t *testing.T) {
	script := healthMetadataScript()
	for _, want := range []string{
		"FTB_HEALTH_PING|OK",
		"FTB_HEALTH_TABLES|",
		"FTB_HEALTH_INDEXES|",
		"FTB_HEALTH_CONSTRAINTS|",
		"FTB_HEALTH_INACTIVE_INDEX|",
		"FTB_HEALTH_INACTIVE_CONSTRAINT|",
		"PRIMARY KEY",
		"UNIQUE",
		"RDB$INDEX_INACTIVE",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("health metadata script missing %q", want)
		}
	}
	if strings.Contains(strings.ToUpper(script), "COALESCE(") {
		t.Fatal("health metadata gate must remain compatible with Firebird 1.0 and avoid COALESCE")
	}
}

func TestHealthReportMarksProductionReadiness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "RELATORIO_SAUDE_PERFORMANCE.txt")
	h := postRecoveryHealth{
		Checked:            true,
		DatabasePath:       `C:\\Redsis\\DBCOM50.RED`,
		Version:            "5.0",
		ConnectivityOK:     true,
		SweepOK:            true,
		ValidationOK:       true,
		Validation:         recoveryStats{},
		UserTableCount:     42,
		UserIndexCount:     100,
		PrimaryUniqueCount: 20,
		Readiness:          "APTO PARA PRODUÇÃO",
	}
	if err := writeHealthReport(path, h, false, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"RELATÓRIO DE SAÚDE", "APTO PARA PRODUÇÃO", "Sweep final concluído: true", "Índices inativos: 0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("health report missing %q:\n%s", want, text)
		}
	}
}

func TestAdvancedRecoveryRunsPostRecoveryHealthGate(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"runPostRecoveryHealth(ctx, c, healthDB, healthReference, healthReportPath, &report)",
		"RELATORIO_SAUDE_PERFORMANCE.txt",
		"NÃO APTO PARA PRODUÇÃO",
		"APTO PARA PRODUÇÃO",
		"setProgress(99, \"Pós-recuperação aprovada\")",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("post-recovery health gate missing %q", want)
		}
	}
}

func TestHelpCoversPostRecoveryHealth(t *testing.T) {
	text := helpText()
	for _, want := range []string{"PÓS-RECUPERAÇÃO / SAÚDE", "RELATORIO_SAUDE_PERFORMANCE.txt", "APTO PARA PRODUÇÃO", "NÃO APTO PARA PRODUÇÃO", "PRIMARY KEY/UNIQUE inativos"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help missing post-recovery concept %q", want)
		}
	}
}

func TestUniqueRelationsFromInactiveIndexes(t *testing.T) {
	details := []inactiveIndexDetail{
		{IndexName: "RDB$PRIMARY12", Relation: "DOCIT"},
		{IndexName: "IDX_DOCIT", Relation: "DOCIT"},
		{IndexName: "IDX_OTHER", Relation: "OUTRA"},
		{IndexName: "SYS", Relation: "RDB$RELATIONS"},
	}
	got := uniqueRelationsFromInactive(details)
	if len(got) != 2 || got[0] != "DOCIT" || got[1] != "OUTRA" {
		t.Fatalf("unexpected relation list: %#v", got)
	}
}

func TestCleanRelationBackupPathsAreDeterministic(t *testing.T) {
	base := `C:\\RECOVERY\\DBCOM_RECOVERY.BAK`
	if got := cleanRelationBaseBackupPath(base, "DOCIT", 2); !strings.Contains(got, "REBUILD_DOCIT_BASE_02") {
		t.Fatalf("unexpected clean base backup path: %s", got)
	}
	if got := cleanRelationBackupPath(base, "DOCIT", 2); !strings.Contains(got, "REBUILD_DOCIT_02") {
		t.Fatalf("unexpected clean backup path: %s", got)
	}
}

func TestHelpTextCoversIterativeConflictAndCleanRebuild(t *testing.T) {
	text := helpText()
	for _, want := range []string{"OUTRA chave duplicada", "continua em loop", "fallback de reconstrução limpa", "leitura NATURAL"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help text missing iterative recovery concept %q", want)
		}
	}
}

func TestPortableDefaultsUseStandardFirebirdPort(t *testing.T) {
	for _, version := range versions {
		if got := defaults[version].Port; got != "3050" {
			t.Fatalf("Firebird %s fallback port must be portable 3050, got %s", version, got)
		}
		if !serviceLooksAutomatic(defaults[version].Service) {
			t.Fatalf("Firebird %s must not depend on a fixed Windows service name: %q", version, defaults[version].Service)
		}
	}
}

func TestParseRegistryFirebirdRootsSupportsNamedInstances(t *testing.T) {
	output := `HKEY_LOCAL_MACHINE\SOFTWARE\Firebird Project\Firebird Server\Instances
    DefaultInstance    REG_SZ    C:\Program Files\Firebird\Firebird_5_0
    Loja               REG_EXPAND_SZ    D:\Servidores\FB25
`
	got := parseRegistryFirebirdRoots(output)
	if len(got) != 2 {
		t.Fatalf("unexpected registry roots: %#v", got)
	}
	if !strings.EqualFold(got[0], `C:\Program Files\Firebird\Firebird_5_0`) || !strings.EqualFold(got[1], `D:\Servidores\FB25`) {
		t.Fatalf("unexpected parsed registry roots: %#v", got)
	}
}

func TestParseFirebirdVersionOutput(t *testing.T) {
	cases := map[string]string{
		"gbak version WI-V5.0.2.1613 Firebird 5.0":  "5.0",
		"gbak version WI-V2.5.9.27139 Firebird 2.5": "2.5",
		"Firebird 3.0": "3.0",
	}
	for input, want := range cases {
		if got := parseFirebirdVersionOutput(input); got != want {
			t.Fatalf("parseFirebirdVersionOutput(%q)=%q want %q", input, got, want)
		}
	}
}

func TestServiceImagePathDiscoveryDoesNotRequireServiceName(t *testing.T) {
	output := `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\BancoPrincipal
    ImagePath    REG_EXPAND_SZ    "D:\Apps\FB5\firebird.exe" -s

HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\OutroServico
    ImagePath    REG_EXPAND_SZ    C:\Windows\System32\svchost.exe -k netsvcs
`
	services := parseWindowsServiceImagePaths(output)
	if len(services) != 2 {
		t.Fatalf("unexpected service parse: %#v", services)
	}
	if services[0].Name != "BancoPrincipal" {
		t.Fatalf("custom service name was not preserved: %#v", services[0])
	}
	if got := firebirdRootFromServiceImage(services[0].ImagePath); !strings.EqualFold(got, `D:\Apps\FB5`) {
		t.Fatalf("unexpected Firebird service root: %q", got)
	}
	if got := firebirdRootFromServiceImage(services[1].ImagePath); got != "" {
		t.Fatalf("non-Firebird service must be ignored, got %q", got)
	}
}

func TestUniversalDistributionIsDocumentedInReleaseWorkflow(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"GOARCH = '386'", "x86_x64", "FirebirdToolbox_${tag}_x86_x64.exe"} {
		if !strings.Contains(text, want) {
			t.Fatalf("release workflow missing universal distribution marker %q", want)
		}
	}
}

func TestParseFirebirdConfigPortPrefersConfiguredValues(t *testing.T) {
	if got := parseFirebirdConfigPort("# RemoteServicePort = 9999\r\nRemoteServicePort = 3055\r\n"); got != "3055" {
		t.Fatalf("unexpected configured port: %q", got)
	}
	if got := parseFirebirdConfigPort("RemoteServiceName = gds_db\r\nRemoteServicePort = 3999\r\n"); got != "3050" {
		t.Fatalf("RemoteServiceName must take precedence, got %q", got)
	}
}

func TestPortFromServiceImagePath(t *testing.T) {
	cases := map[string]string{
		`"C:\Firebird\firebird.exe" -s -p 3057`: "3057",
		`C:\Firebird\fbserver.exe -p=3060`:      "3060",
		`C:\Firebird\firebird.exe -s`:           "",
	}
	for input, want := range cases {
		if got := portFromServiceImagePath(input); got != want {
			t.Fatalf("portFromServiceImagePath(%q)=%q want %q", input, got, want)
		}
	}
}

func TestHealthRowCountScriptIsDialect1SafeAndNatural(t *testing.T) {
	script := healthRowCountScript("DOCIT")
	for _, want := range []string{"SET LIST ON", "COUNT(*) AS FTB_HEALTH_ROWCOUNT", "FROM DOCIT T", "PLAN (T NATURAL)"} {
		if !strings.Contains(script, want) {
			t.Fatalf("health row count script missing %q:\n%s", want, script)
		}
	}
	for _, forbidden := range []string{"CAST(COUNT(*)", "FTB_HEALTH_ROWCOUNT|", `FROM "DOCIT"`} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("health row count script reintroduced Dialect 1 incompatibility %q:\n%s", forbidden, script)
		}
	}
}

func TestHealthRowCountIdentifierPreservesDelimitedModernNames(t *testing.T) {
	if got := healthRowCountIdentifier("CaseSensitiveTable"); got != `"CaseSensitiveTable"` {
		t.Fatalf("unexpected delimited identifier: %q", got)
	}
}

func TestParseHealthRowCountListOutput(t *testing.T) {
	out := "Database: test, User: SYSDBA\nSQL> \nFTB_HEALTH_ROWCOUNT              203045\nSQL>"
	n, ok := parseHealthRowCount(out)
	if !ok || n != 203045 {
		t.Fatalf("unexpected parsed row count: %d ok=%v", n, ok)
	}
}

func TestVerifiedRestoreSurvivesUntilPostRecoveryHealth(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"Restore de teste preservado temporariamente para comparação pós-recuperação",
		"cleanupTestRestoreAfterHealth(testPath, &report)",
		"Restore de referência preservado porque um gate de pós-recuperação falhou",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("restore-reference lifecycle missing %q", want)
		}
	}
}

func TestResolveResultFolderPrefersExistingResultOverSuggestedDestination(t *testing.T) {
	root := t.TempDir()
	destDir := filepath.Join(root, "dest")
	reportDir := filepath.Join(root, "recovery")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(reportDir, "RELATORIO_RECUPERACAO.txt")
	if err := os.WriteFile(report, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	// O destino ainda não existe, embora a pasta pai exista. Nesse cenário o
	// relatório real produzido pela rotina deve vencer o destino apenas sugerido.
	dest := filepath.Join(destDir, "DBCOM50.RED")
	if got := resolveResultFolder(dest, report); got != reportDir {
		t.Fatalf("resolveResultFolder() = %q, want %q", got, reportDir)
	}
}

func TestResolveResultFolderPrefersGeneratedDestinationWhenItExists(t *testing.T) {
	root := t.TempDir()
	destDir := filepath.Join(root, "dest")
	reportDir := filepath.Join(root, "recovery")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(destDir, "DBCOM50.RED")
	if err := os.WriteFile(dest, []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(reportDir, "RELATORIO_RECUPERACAO.txt")
	if err := os.WriteFile(report, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := resolveResultFolder(dest, report); got != destDir {
		t.Fatalf("resolveResultFolder() = %q, want %q", got, destDir)
	}
}

func TestResolveResultFolderFallsBackToParentBeforeFirstRun(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "DBCOM50.RED")
	if got := resolveResultFolder(dest); got != root {
		t.Fatalf("resolveResultFolder() = %q, want %q", got, root)
	}
}

func TestStaticLabelsUseSolidWindowBackgroundInsteadOfTransparentBrush(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"staticBrushWhite",
		"procIsWindowEnabled.Call(lParam)",
		"procSetBkColor.Call(wParam, rgb(255, 255, 255))",
		"return uintptr(staticBrushWhite)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("solid static background regression: missing %q", want)
		}
	}
	if strings.Contains(text, "nullBrush, _, _ := procGetStockObject.Call(NULL_BRUSH)") {
		t.Fatal("transparent NULL_BRUSH path reintroduced; it causes stale/duplicated labels on repaint")
	}
}

func TestPreflightGuardsLongRunningRecovery(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"runPreflight(c)",
		"ensureWritableDir",
		"diskFreeBytes",
		"GetDiskFreeSpaceExW",
		"probeFirebirdVersion",
		"portReachable",
		"Pré-validação reprovada",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("preflight regression: missing %q", want)
		}
	}
}

func TestPreferencesNeverPersistPassword(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	start := strings.Index(text, "type appPreferences struct")
	end := strings.Index(text[start:], "func preferencesPath")
	if start < 0 || end < 0 {
		t.Fatal("appPreferences block not found")
	}
	block := text[start : start+end]
	for _, forbidden := range []string{"Password", "password", "Senha", "senha"} {
		if strings.Contains(block, forbidden) {
			t.Fatalf("sensitive preference field reintroduced: %q", forbidden)
		}
	}
	if !strings.Contains(text, `filepath.Join(base, "FirebirdToolbox", "preferences.json")`) {
		t.Fatal("preferences path must stay isolated under the user config directory")
	}
}

func TestProgressAndCancellationAreOperatorFriendly(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`Fase %d/%d • %s`,
		"Cancelamento solicitado — aguardando gbak/gfix encerrar com segurança",
		"aguardando encerramento seguro do processo Firebird",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("operator progress regression: missing %q", want)
		}
	}
}

func TestResultShortcutsRemainAvailable(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`createIconButton(idOpenReport, "Abrir relatório", uiOpenReportX, uiResultShortcutY, uiOpenReportWidth, 28)`,
		`createIconButton(idOpenDatabase, "Localizar banco", uiLocateDatabaseX, uiResultShortcutY, uiLocateDatabaseWidth, 28)`,
		"openResultReport()",
		"openResultDatabase()",
		`"/select," + dest`,
		"fileGeneratedThisRun(dest)",
		"lastDatabaseSnapshot()",
		"Banco final localizado e selecionado no Windows Explorer.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("result shortcut regression: missing %q", want)
		}
	}
}

func TestDPIValidationChecklistCoversCommonWindowsScales(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("docs", "VALIDACAO_DPI.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"100%", "125%", "150%", "175%", "Abrir relatório", "Localizar banco", "Fase X/Y"} {
		if !strings.Contains(text, want) {
			t.Fatalf("DPI validation checklist missing %q", want)
		}
	}
}

func TestMainWindowIconIsAppliedBeforeWindowIsShown(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`procLoadImageW.Call(`,
		`uintptr(1), // MAKEINTRESOURCE(1)`,
		`hIcon:         appIconBig`,
		`hIconSm:       appIconSmall`,
		`setAppIcons()`,
		`procShowWindow.Call(hwnd, SW_SHOW)`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("window/taskbar icon regression: missing %q", want)
		}
	}
	mainWindowStart := strings.Index(text, `uintptr(unsafe.Pointer(u16("Firebird Toolbox")))`)
	if mainWindowStart < 0 {
		t.Fatal("main window creation block not found")
	}
	mainWindowEnd := strings.Index(text[mainWindowStart:], `mainHwnd = syscall.Handle(hwnd)`)
	if mainWindowEnd < 0 {
		t.Fatal("main window creation block end not found")
	}
	block := text[mainWindowStart : mainWindowStart+mainWindowEnd]
	if strings.Contains(block, `WS_VISIBLE`) {
		t.Fatal("main window must be created hidden so the taskbar never sees it before the app icon is applied")
	}
}
