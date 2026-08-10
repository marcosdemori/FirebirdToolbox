//go:build windows

package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

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
	want := []string{"Corrigir banco", "Corrigir banco (avançado)", "Somente restaurar"}
	if len(routines) != len(want) {
		t.Fatalf("unexpected routine count: got %d want %d", len(routines), len(want))
	}
	for i := range want {
		if routines[i].Name != want[i] {
			t.Fatalf("routine[%d] = %q, want %q", i, routines[i].Name, want[i])
		}
		if strings.Contains(strings.ToLower(routines[i].Name), "converter") {
			t.Fatalf("converter routine must not be exposed: %q", routines[i].Name)
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
	if appVersion != "1.2.10" {
		t.Fatalf("unexpected app version: %s", appVersion)
	}
	if appCredit != "Projeto feito por Marcos De Mori Laiola" {
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
	if uiSkipHintWidth < 290 {
		t.Fatalf("skip_data help label is too narrow: %d", uiSkipHintWidth)
	}
	if uiSkipHintX+uiSkipHintWidth > 884 {
		t.Fatalf("skip_data help label exceeds the execution area: x=%d width=%d", uiSkipHintX, uiSkipHintWidth)
	}
	if uiSkipEditX+uiSkipEditWidth+12 > uiSkipHintX {
		t.Fatalf("skip_data edit and help label overlap: editEnd=%d hintX=%d", uiSkipEditX+uiSkipEditWidth, uiSkipHintX)
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
		"Corrigir banco",
		"Corrigir banco (avançado)",
		"Somente restaurar",
		"PULAR TABELAS (REGEX)",
		"RECUPERAÇÃO PARCIAL",
		"RELATORIO_RECUPERACAO.txt",
		"LOG_TECNICO_RECUPERACAO.txt",
		"Projeto feito por Marcos De Mori Laiola",
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
	for _, want := range []string{`RDB$DB_KEY`, `PLAN (T NATURAL)`, `IF (N > 1)`, `DELETE FROM "DOCIT"`} {
		if !strings.Contains(script, want) {
			t.Fatalf("cleanup script missing %q:\n%s", want, script)
		}
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
	for _, want := range []string{"PRIMARY KEY", "UNIQUE", "-inactive", "CONFLITOS_DUPLICIDADE.txt", "primeira ocorrência"} {
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
		`"FileVersion": "1.2.10.0"`,
		`"ProductVersion": "1.2.10"`,
		`"execution-level": "as invoker"`,
		`"use-common-controls-v6": true`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("winres metadata missing %q", want)
		}
	}
}
