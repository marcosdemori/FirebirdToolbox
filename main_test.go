//go:build windows

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

var (
	procGetDC                 = user32.NewProc("GetDC")
	procReleaseDC             = user32.NewProc("ReleaseDC")
	procGetTextExtentPoint32W = gdi32.NewProc("GetTextExtentPoint32W")
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
	// o resumo so cita backup, destino e relatorio que existam de fato em disco
	dir := t.TempDir()
	backup := filepath.Join(dir, "DBCOM_20260810_101539.BAK")
	report := filepath.Join(dir, "RELATORIO_RECUPERACAO.txt")
	dest := filepath.Join(dir, "DBCOM50.RED")
	for _, path := range []string{backup, report, dest} {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("could not create %s: %v", path, err)
		}
	}
	setLastHeader(databaseHeader{ODS: "13.1", Dialect: "1", PageSize: "4096"})
	setLastBackup(backup)
	setLastReport(report)
	c := workflowConfig{
		Routine:       routines[0],
		TargetVersion: "5.0",
		Dest:          dest,
	}
	text, compact := successSummary(c)
	for _, want := range []string{"Rotina concluída com sucesso", "Firebird de destino: 5.0", "ODS 13.1", "Dialeto 1", dest, "RELATORIO_RECUPERACAO.txt"} {
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
	args := restoreTestArgs("5.0", "3055", "SYSDBA", "secret", "backup", "dest", true, false)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-one_at_a_time") {
		t.Fatalf("restore test args must include -one_at_a_time: %v", args)
	}
	if strings.Contains(joined, " -n ") {
		t.Fatalf("restore test args must not drop constraints unless asked: %v", args)
	}
}

func TestRestoreTestArgsNoValidity(t *testing.T) {
	args := restoreTestArgs("5.0", "3055", "SYSDBA", "secret", "backup", "dest", false, true)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, " -n ") {
		t.Fatalf("restore test args must include -n when no_validity is requested: %v", args)
	}
	if strings.Contains(joined, "-one_at_a_time") {
		t.Fatalf("-n must not imply -one_at_a_time: %v", args)
	}
}

func TestGbakLooksLikeOrphanForeignKey(t *testing.T) {
	output := "gbak: ERROR:action cancelled by trigger (2) to preserve data integrity\n" +
		"gbak: ERROR:    Non-existent PRIMARY or UNIQUE KEY specified for FOREIGN KEY."
	if !gbakLooksLikeOrphanForeignKey(output) {
		t.Fatal("restore failure on foreign key recreation must be detected")
	}
	if gbakLooksLikeOrphanForeignKey("gbak: ERROR:    expected index B-tree") {
		t.Fatal("index corruption must not be reported as broken referential integrity")
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
	if appVersion != "1.2.32" {
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

// A descricao da rotina e um STATIC de uma linha: texto mais largo que o
// controle e cortado em silencio, sem reticencias e sem quebrar linha. Mede
// cada descricao com a mesma fonte da interface para impedir que uma nova
// redacao volte a estourar a largura disponivel.
func TestRoutineDescriptionsFitTheirControl(t *testing.T) {
	font := createFont(-15, FW_NORMAL, "Segoe UI")
	if font == 0 {
		t.Skip("CreateFontW indisponivel neste ambiente")
	}
	defer procDeleteObject.Call(uintptr(font))

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		t.Skip("GetDC indisponivel neste ambiente")
	}
	defer procReleaseDC.Call(0, hdc)

	previous, _, _ := procSelectObject.Call(hdc, uintptr(font))
	defer procSelectObject.Call(hdc, previous)

	for _, routine := range routines {
		text := utf16.Encode([]rune(routine.Description))
		var size struct{ CX, CY int32 }
		ok, _, _ := procGetTextExtentPoint32W.Call(
			hdc,
			uintptr(unsafe.Pointer(&text[0])),
			uintptr(len(text)),
			uintptr(unsafe.Pointer(&size)),
		)
		if ok == 0 {
			t.Fatalf("nao foi possivel medir a descricao de %q", routine.Name)
		}
		if size.CX > uiDescriptionWidth {
			t.Fatalf("descricao de %q ocupa %d px e seria cortada em %d px: %s",
				routine.Name, size.CX, uiDescriptionWidth, routine.Description)
		}
	}
}

// Caso real: DBCOM.RED era ODS 13.0 (Firebird 4.0) e a rotina estava
// configurada para Firebird 2.5, cujo gfix so abre ate ODS 11.2.
const unsupportedODSOutput = "unsupported on-disk structure for file C:\\REDSIS\\CLIENTES\\CS STONE\\DBCOM.RED; found 13.0, support 11.2"

func TestParseUnsupportedODSReadsFirebirdRefusal(t *testing.T) {
	found, support := parseUnsupportedODS(unsupportedODSOutput)
	if found != "13.0" || support != "11.2" {
		t.Fatalf("ODS lido incorretamente: found=%q support=%q", found, support)
	}
	if f, sup := parseUnsupportedODS("gfix: database validated"); f != "" || sup != "" {
		t.Fatalf("saida sem recusa nao pode virar incompatibilidade: found=%q support=%q", f, sup)
	}
}

func TestUnsupportedODSMessageNamesTheVersionToSelect(t *testing.T) {
	found, support := parseUnsupportedODS(unsupportedODSOutput)
	message := unsupportedODSMessage("2.5", found, support)
	for _, want := range []string{"ODS 13.0", "Firebird 4.0", "ODS 11.2", "Versao da rotina"} {
		if !strings.Contains(message, want) {
			t.Fatalf("mensagem de ODS incompativel nao cita %q: %s", want, message)
		}
	}
}

func TestODSMapCoversEverySupportedVersion(t *testing.T) {
	mapped := map[string]bool{}
	for _, version := range odsFirebirdVersion {
		mapped[version] = true
	}
	for _, version := range versions {
		if !mapped[version] {
			t.Fatalf("nenhum ODS conhecido aponta para Firebird %s", version)
		}
	}
}

func TestToolFailureSurfacesTheRealCause(t *testing.T) {
	base := errors.New("exit status 1")
	err := toolFailure("o diagnóstico somente leitura não pôde ser concluído", base, unsupportedODSOutput)
	text := err.Error()
	for _, want := range []string{"unsupported on-disk structure", "found 13.0, support 11.2", "exit status 1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("erro perdeu %q: %s", want, text)
		}
	}
	if !errors.Is(err, base) {
		t.Fatal("toolFailure precisa preservar o erro original")
	}
	if detail := toolFailureDetail("COMANDO: gfix.exe -v\n\nfalha real\n"); detail != "falha real" {
		t.Fatalf("toolFailureDetail deveria pular o eco do comando: %q", detail)
	}
}

func TestReportDoesNotClaimCleanDiagnosticWhenItNeverRan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "RELATORIO_RECUPERACAO.txt")
	r := recoveryReport{
		ReportPath:    path,
		Source:        `C:\Redsis\DBCOM.RED`,
		SourceVersion: "2.5",
		Started:       time.Now(),
		Finished:      time.Now(),
		FinalState:    "FALHA / RECUPERAÇÃO NÃO CONCLUÍDA",
	}
	r.fail("o diagnóstico somente leitura não pôde ser concluído")
	if err := writeRecoveryReport(r); err != nil {
		t.Fatalf("writeRecoveryReport: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "nenhuma corrupção estrutural foi reportada") {
		t.Fatalf("relatório afirma diagnóstico limpo sem diagnóstico: %s", text)
	}
	if !strings.Contains(text, "DIAGNÓSTICO NÃO CONCLUÍDO") {
		t.Fatalf("relatório não avisa que o diagnóstico não rodou: %s", text)
	}

	r.DiagnosticDone = true
	if err := writeRecoveryReport(r); err != nil {
		t.Fatalf("writeRecoveryReport: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "nenhuma corrupção estrutural foi reportada") {
		t.Fatalf("diagnóstico concluído e limpo deve continuar sendo informado: %s", data)
	}
}

// Caso real: o gbak morreu logo depois de gravar EMAIL, mas quem quebrou foi
// CAIXA, como o firebird.log registrou duas vezes.
const bugcheckFirebirdLog = `MARCOS	Thu Oct 15 17:05:49 2026
	Database: C:\\REDSIS\\CLIENTES\\CS STONE\\DBCOM_WORK.RED
	Error: Record 2594350 has bad transaction 41437787 in table CAIXA (132)

MARCOS	Thu Oct 15 17:05:49 2026
	bugcheck during scan of table 132 (CAIXA)

MARCOS	Thu Oct 15 17:05:50 2026
	Database: C:\\REDSIS\\CLIENTES\\CS STONE\\DBCOM_WORK.RED
	Validation aborted
	database file appears corrupt
	wrong page type
	page 386932 is of wrong type (expected transaction inventory, found index B-tree)
`

const bugcheckGbakOutput = `gbak:    writing data for table EMAIL
gbak:10000 records written
gbak: ERROR:database file appears corrupt (C:\\REDSIS\\CLIENTES\\CS STONE\\DBCOM_WORK.RED)
gbak: ERROR:    wrong page type
gbak: ERROR:    page 413591 is of wrong type (expected transaction inventory, found data)
gbak: ERROR:gds_$receive failed
gbak:Exiting before completion due to errors
`

// Os dois logs respondem perguntas diferentes e nao podem ser trocados um pelo
// outro. Prova medida em producao no DBCOM.RED: com -skip_data em CAIXA, o
// sinal do firebird.log, o gbak travou na mesma pagina 413591 e no mesmo
// registro 16121 do backup sem skip; com -skip_data em EMAIL, o sinal do gbak,
// o backup passou de 413591 e so parou adiante, em outro sitio de dano.
func TestEachLogAnswersItsOwnQuestion(t *testing.T) {
	// "Onde o motor quebrou" - diagnostico, vem do firebird.log.
	if got := suspectedRelationFromFirebirdLog(bugcheckFirebirdLog); got != "CAIXA" {
		t.Fatalf("o bugcheck do firebird.log deveria apontar CAIXA, veio %q", got)
	}
	// "Qual relacao bloqueia o backup" - alvo do -skip_data, vem do gbak.
	if got := suspectedRelationFromGbak(bugcheckGbakOutput); got != "EMAIL" {
		t.Fatalf("o log do gbak deveria apontar EMAIL, veio %q", got)
	}
	if got := suspectedRelationFromFirebirdLog("Validation started\nValidation finished"); got != "" {
		t.Fatalf("log sem bugcheck nao pode gerar suspeita: %q", got)
	}
}

func TestReportSeparatesSkipTargetFromEngineCrash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "RELATORIO_RECUPERACAO.txt")
	r := recoveryReport{
		ReportPath:              path,
		Source:                  `C:\Redsis\DBCOM.RED`,
		SourceVersion:           "5.0",
		Started:                 time.Now(),
		Finished:                time.Now(),
		DiagnosticDone:          true,
		SuspectedRelation:       suspectedRelationFromGbak(bugcheckGbakOutput),
		SuspectedRelationSource: "log do gbak",
		EngineCrashRelation:     suspectedRelationFromFirebirdLog(bugcheckFirebirdLog),
		FinalState:              "NÃO RECUPERÁVEL COM FERRAMENTAS PADRÃO",
	}
	if r.SuspectedRelation != "EMAIL" || r.EngineCrashRelation != "CAIXA" {
		t.Fatalf("montagem do caso errada: skip=%q bugcheck=%q", r.SuspectedRelation, r.EngineCrashRelation)
	}
	if err := writeRecoveryReport(r); err != nil {
		t.Fatalf("writeRecoveryReport: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"Possível relação afetada: EMAIL (identificada pelo log do gbak)",
		"Tabela onde o motor sofreu bugcheck: CAIXA (firebird.log)",
		"As duas tabelas são diferentes",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("relatório não separa os dois sinais, falta %q: %s", want, text)
		}
	}
}

func TestIndexCorruptionNeedsIndexOnTheExpectedSide(t *testing.T) {
	// Pagina de indice realmente danificada: o indice esta no lado esperado.
	if !gbakLooksLikeIndexCorruption("page 213030 is of wrong type (expected index B-tree, found blob)") {
		t.Fatal("corrupcao real de indice precisa continuar sendo detectada")
	}
	if !gbakLooksLikeIndexCorruption("gbak: ERROR: index page errors") {
		t.Fatal("index page errors precisa continuar sendo detectado")
	}
	// Caso real: a pagina danificada e de transaction inventory. O indice so
	// aparece como invasor, ou nas linhas "writing index ..." do proprio gbak.
	if gbakLooksLikeIndexCorruption(bugcheckGbakOutput) {
		t.Fatal("falha de transaction inventory nao pode ser classificada como corrupcao de indice")
	}
	if gbakLooksLikeIndexCorruption("page 386932 is of wrong type (expected transaction inventory, found index B-tree)") {
		t.Fatal("indice encontrado no lugar errado nao significa indice danificado")
	}
	if gbakLooksLikeIndexCorruption("gbak:    writing index RDB$PRIMARY214\ngbak: ERROR:    wrong page type") {
		t.Fatal("a mencao rotineira a \"writing index\" nao pode virar suspeita de indice")
	}
}

// Guarda de fiacao: ja foi tentado uma vez ligar o alvo do -skip_data ao
// bugcheck do firebird.log, e o resultado medido em producao foi o backup
// travar no mesmo ponto de antes. SuspectedRelation comanda -skip_data, resgate
// por indices e resgate por registros, entao ela so pode vir do log do gbak.
func TestSkipDataTargetComesFromTheGbakLog(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "report.SuspectedRelation = suspectedRelationFromGbak(backupOutput)") {
		t.Fatal("a relação do -skip_data precisa vir de suspectedRelationFromGbak")
	}
	if strings.Contains(text, "report.SuspectedRelation = suspectedRelationFromFirebirdLog") {
		t.Fatal("o bugcheck do firebird.log não pode escolher o alvo do -skip_data; ele é diagnóstico e vai para EngineCrashRelation")
	}
	if !strings.Contains(text, "report.EngineCrashRelation = suspectedRelationFromFirebirdLog") {
		t.Fatal("o bugcheck do firebird.log precisa continuar sendo registrado em EngineCrashRelation")
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
		`"FileVersion": "1.2.32.0"`,
		`"ProductVersion": "1.2.32"`,
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

func TestParseConstraintDefinitionsGroupsCompositeForeignKey(t *testing.T) {
	output := strings.Join([]string{
		"FTB_FKC|FK_ITEM_PEDIDO|ITEM|COD_EMPRESA|PEDIDO|COD_EMPRESA|CASCADE|CASCADE",
		"FTB_FKC|FK_ITEM_PEDIDO|ITEM|NUM_PEDIDO|PEDIDO|NUM_PEDIDO|CASCADE|CASCADE",
		"FTB_FKC|FK_ITEM_PRODUTO|ITEM|COD_PRODUTO|PRODUTO|CODIGO||RESTRICT",
		"FTB_CHK|CHK_QTD|ITEM|14|CHECK (QTD > 0)",
		"FTB_CHK|CHK_QTD|ITEM|14|CHECK (QTD > 0)",
	}, "\r\n")
	got := parseConstraintDefinitions(output)
	if len(got) != 3 {
		t.Fatalf("expected 3 constraints, got %d: %#v", len(got), got)
	}
	composite := got[0]
	if composite.Name != "FK_ITEM_PEDIDO" || composite.Relation != "ITEM" || composite.RefRelation != "PEDIDO" {
		t.Fatalf("unexpected composite FK header: %#v", composite)
	}
	if len(composite.Columns) != 2 || composite.Columns[0] != "COD_EMPRESA" || composite.Columns[1] != "NUM_PEDIDO" {
		t.Fatalf("composite FK must keep both child columns in index order: %#v", composite.Columns)
	}
	if len(composite.RefColumns) != 2 || composite.RefColumns[1] != "NUM_PEDIDO" {
		t.Fatalf("composite FK must keep both parent columns in index order: %#v", composite.RefColumns)
	}
	if composite.UpdateRule != "CASCADE" || composite.DeleteRule != "CASCADE" {
		t.Fatalf("referential rules lost: %#v", composite)
	}
	check := got[2]
	if check.Type != constraintTypeCheck || check.Name != "CHK_QTD" || check.Check != "CHECK (QTD > 0)" {
		t.Fatalf("unexpected CHECK constraint: %#v", check)
	}
}

func TestBuildConstraintsDDLScriptRecreatesForeignKeyAndCheck(t *testing.T) {
	script := buildConstraintsDDLScript([]constraintDefinition{
		{
			Name: "FK_ITEM_PEDIDO", Relation: "ITEM", Type: constraintTypeForeignKey,
			Columns: []string{"COD_EMPRESA", "NUM_PEDIDO"}, RefRelation: "PEDIDO", RefColumns: []string{"COD_EMPRESA", "NUM_PEDIDO"},
			UpdateRule: "CASCADE", DeleteRule: "RESTRICT",
		},
		{Name: "CHK_QTD", Relation: "ITEM", Type: constraintTypeCheck, Check: "CHECK (QTD > 0)"},
		{Name: "FK_QUEBRADA", Relation: "ITEM", Type: constraintTypeForeignKey, Columns: []string{"A"}, RefRelation: "PEDIDO"},
	}, "3")
	wantFK := `ALTER TABLE "ITEM" ADD CONSTRAINT "FK_ITEM_PEDIDO" FOREIGN KEY ("COD_EMPRESA", "NUM_PEDIDO") REFERENCES "PEDIDO" ("COD_EMPRESA", "NUM_PEDIDO") ON UPDATE CASCADE;`
	if !strings.Contains(script, wantFK) {
		t.Fatalf("foreign key DDL missing or wrong.\nwant: %s\ngot:\n%s", wantFK, script)
	}
	if strings.Contains(script, "ON DELETE RESTRICT") {
		t.Fatal("RESTRICT is the default rule and must not be emitted")
	}
	wantCheck := `ALTER TABLE "ITEM" ADD CONSTRAINT "CHK_QTD" CHECK (QTD > 0);`
	if !strings.Contains(script, wantCheck) {
		t.Fatalf("check constraint DDL missing.\nwant: %s\ngot:\n%s", wantCheck, script)
	}
	if strings.Contains(script, "FK_QUEBRADA") {
		t.Fatal("a foreign key without matching parent columns must be skipped, not emitted broken")
	}
	if !strings.Contains(script, "COMMIT;") {
		t.Fatal("script must commit each constraint")
	}
}

func TestBuildConstraintsDDLScriptWithoutConstraintsStaysExecutable(t *testing.T) {
	script := buildConstraintsDDLScript(nil, "3")
	if strings.Contains(script, "ALTER TABLE") {
		t.Fatalf("empty input must not produce DDL: %s", script)
	}
	if !strings.Contains(script, "/*") {
		t.Fatalf("empty script must explain itself in a comment: %s", script)
	}
}

func TestBuildOrphanRowsScriptOnlySelects(t *testing.T) {
	script := buildOrphanRowsScript([]constraintDefinition{
		{
			Name: "FK_ITEM_PEDIDO", Relation: "ITEM", Type: constraintTypeForeignKey,
			Columns: []string{"COD_EMPRESA", "NUM_PEDIDO"}, RefRelation: "PEDIDO", RefColumns: []string{"COD_EMPRESA", "NUM_PEDIDO"},
		},
		{Name: "CHK_QTD", Relation: "ITEM", Type: constraintTypeCheck, Check: "CHECK (QTD > 0)"},
	}, "3")
	var statements []string
	for _, line := range strings.Split(script, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "/*") {
			continue
		}
		statements = append(statements, strings.ToUpper(line))
	}
	executable := strings.Join(statements, "\n")
	for _, forbidden := range []string{"DELETE", "UPDATE", "ALTER", "DROP", "INSERT"} {
		if strings.Contains(executable, forbidden) {
			t.Fatalf("orphan inspection script must never change data, found %q:\n%s", forbidden, script)
		}
	}
	want := `SELECT F.* FROM "ITEM" F WHERE F."COD_EMPRESA" IS NOT NULL AND F."NUM_PEDIDO" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "PEDIDO" P WHERE P."COD_EMPRESA" = F."COD_EMPRESA" AND P."NUM_PEDIDO" = F."NUM_PEDIDO");`
	if !strings.Contains(script, want) {
		t.Fatalf("orphan query missing or wrong.\nwant: %s\ngot:\n%s", want, script)
	}
	if !strings.Contains(script, `SELECT COUNT(*) FROM "ITEM" F WHERE`) {
		t.Fatalf("orphan script must count the orphan rows first:\n%s", script)
	}
	if strings.Contains(script, "CHK_QTD") {
		t.Fatal("CHECK constraints have no parent rows to inspect and must be skipped")
	}
}

func TestBuildOrphanRowsScriptWithoutForeignKeys(t *testing.T) {
	script := buildOrphanRowsScript(nil, "3")
	if strings.Contains(script, "SELECT") {
		t.Fatalf("empty input must not produce queries: %s", script)
	}
}

func TestConstraintsExtractionScriptReadsSystemTables(t *testing.T) {
	script := constraintsExtractionScript()
	for _, want := range []string{
		"RDB$RELATION_CONSTRAINTS",
		"RDB$REF_CONSTRAINTS",
		"RDB$INDEX_SEGMENTS",
		"RDB$CHECK_CONSTRAINTS",
		"RDB$TRIGGERS",
		"FTB_FKC|",
		"FTB_CHK|",
		"QUIT;",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("constraint extraction script missing %q", want)
		}
	}
}

func TestNoValidityWarningsPointToTheGeneratedScripts(t *testing.T) {
	if got := noValidityConstraintWarnings(recoveryReport{}); got != nil {
		t.Fatalf("a normal restore must not emit -n warnings: %#v", got)
	}
	joined := strings.Join(noValidityConstraintWarnings(recoveryReport{
		NoValidityRestore:    true,
		ConstraintsScript:    `C:\BKP\RECOVERY_X\CONSTRAINTS_ORIGINAIS.sql`,
		ConstraintsExtracted: 7,
		OrphanRowsScript:     `C:\BKP\RECOVERY_X\LINHAS_ORFAS.sql`,
	}), "\n")
	for _, want := range []string{
		"NÃO existem no banco restaurado",
		`C:\BKP\RECOVERY_X\CONSTRAINTS_ORIGINAIS.sql`,
		`C:\BKP\RECOVERY_X\LINHAS_ORFAS.sql`,
		"apto para produção",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("-n warnings missing %q:\n%s", want, joined)
		}
	}
	fallback := strings.Join(noValidityConstraintWarnings(recoveryReport{NoValidityRestore: true}), "\n")
	if !strings.Contains(fallback, "backup anterior") {
		t.Fatalf("without any extracted DDL the report must say where else to get it:\n%s", fallback)
	}
}

func TestWindowsBuildScriptsAreUTF8WithBOM(t *testing.T) {
	// O Windows PowerShell 5.1 le arquivos .ps1 sem BOM usando a codepage ANSI
	// do sistema, o que corrompe cada acento das mensagens e das constantes
	// comparadas pelo verificador. UTF-8 com BOM e o unico encoding que o 5.1 e
	// o PowerShell 7 do CI interpretam da mesma forma.
	paths := []string{
		"tools/build.ps1",
		"tools/prepare-resources.ps1",
		"tools/verify-windows-build.ps1",
		"tools/sign-windows.ps1",
		"tools/defender-preflight.ps1",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
			t.Fatalf("%s must start with the UTF-8 BOM so Windows PowerShell 5.1 reads it as UTF-8", path)
		}
		if !utf8.Valid(data) {
			t.Fatalf("%s is not valid UTF-8", path)
		}
	}
}

func TestWindowsBuildScriptsDecideNativeCommandsByExitCode(t *testing.T) {
	// Com $ErrorActionPreference = "Stop", o Windows PowerShell 5.1 transforma a
	// saida de stderr de um comando nativo (por exemplo "go: downloading ...")
	// em NativeCommandError terminante mesmo com exit code 0. As chamadas a go e
	// go-winres precisam passar pelo wrapper que decide pelo $LASTEXITCODE.
	paths := []string{
		"tools/build.ps1",
		"tools/prepare-resources.ps1",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(data)
		if !strings.Contains(text, "function Invoke-NativeCommand") {
			t.Fatalf("%s must define Invoke-NativeCommand to survive native stderr on PowerShell 5.1", path)
		}
		if !strings.Contains(text, `$ErrorActionPreference = "Stop"`) {
			t.Fatalf("%s must keep $ErrorActionPreference = \"Stop\" for the rest of the script", path)
		}
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			for _, bare := range []string{"go ", "gofmt ", "go-winres ", "& go ", "& gofmt ", "& go-winres "} {
				if strings.HasPrefix(trimmed, bare) {
					t.Fatalf("%s invokes a native command directly (%q); use Invoke-NativeCommand", path, trimmed)
				}
			}
		}
	}
}

func TestParseConstraintDefinitionsFlagsTruncatedCheckSource(t *testing.T) {
	short := "CHECK (QTD > 0)"
	long := "CHECK (" + strings.Repeat("A", constraintCheckSourceLimit) + ")"
	output := strings.Join([]string{
		"FTB_CHK|CHK_CURTA|ITEM|" + strconv.Itoa(len(short)) + "|" + short,
		"FTB_CHK|CHK_LONGA|ITEM|" + strconv.Itoa(constraintCheckSourceLimit+120) + "|" + long,
	}, "\r\n")
	got := parseConstraintDefinitions(output)
	if len(got) != 2 {
		t.Fatalf("expected 2 CHECK constraints, got %d: %#v", len(got), got)
	}
	if got[0].Truncated {
		t.Fatalf("a CHECK that fits the CAST must not be flagged: %#v", got[0])
	}
	if !got[1].Truncated {
		t.Fatalf("a CHECK longer than the CAST limit must be flagged as truncated: %#v", got[1])
	}
	names := truncatedCheckConstraints(got)
	if len(names) != 1 || !strings.Contains(names[0], "CHK_LONGA") {
		t.Fatalf("truncated CHECK must be reported by name: %#v", names)
	}
}

func TestParseConstraintDefinitionsFlagsTruncationWithoutLength(t *testing.T) {
	// Banco que nao soube informar o CHAR_LENGTH: o proprio texto no limite
	// do CAST denuncia o corte.
	atLimit := strings.Repeat("A", constraintCheckSourceLimit)
	output := "FTB_CHK|CHK_SEM_TAMANHO|ITEM|?|CHECK (" + atLimit
	got := parseConstraintDefinitions(output)
	if len(got) != 1 {
		t.Fatalf("expected 1 CHECK constraint, got %d", len(got))
	}
	if !got[0].Truncated {
		t.Fatalf("source at the CAST limit must be treated as truncated: %#v", got[0])
	}
}

func TestBuildConstraintsDDLScriptCommentsOutTruncatedCheck(t *testing.T) {
	script := buildConstraintsDDLScript([]constraintDefinition{
		{Name: "CHK_LONGA", Relation: "ITEM", Type: constraintTypeCheck, Check: "CHECK (QTD > 0 AND DESCRICAO", Truncated: true},
		{Name: "CHK_OK", Relation: "ITEM", Type: constraintTypeCheck, Check: "CHECK (QTD > 0)"},
	}, "3")
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		if strings.Contains(trimmed, "CHK_LONGA") {
			t.Fatalf("a truncated CHECK must never be emitted as an executable command: %s", trimmed)
		}
	}
	if !strings.Contains(script, `-- ALTER TABLE "ITEM" ADD CONSTRAINT "CHK_LONGA"`) {
		t.Fatalf("the truncated CHECK must stay in the script, commented out:\n%s", script)
	}
	if !strings.Contains(script, "CORTADA") {
		t.Fatalf("the script must say the expression came back truncated:\n%s", script)
	}
	if !strings.Contains(script, `ALTER TABLE "ITEM" ADD CONSTRAINT "CHK_OK" CHECK (QTD > 0);`) {
		t.Fatalf("a complete CHECK must still be executable:\n%s", script)
	}
}

func TestConstraintIdentifiersQuotedFollowsDialect(t *testing.T) {
	if constraintIdentifiersQuoted("1") {
		t.Fatal("dialect 1 treats double quotes as a string literal, so identifiers must stay unquoted")
	}
	for _, dialect := range []string{"3", "", " ", "2"} {
		if !constraintIdentifiersQuoted(dialect) {
			t.Fatalf("dialect %q must keep quoted identifiers", dialect)
		}
	}
}

func TestConstraintScriptsInDialect1UseNoDoubleQuotes(t *testing.T) {
	constraints := []constraintDefinition{
		{
			Name: "FK_ITEM_PEDIDO", Relation: "ITEM", Type: constraintTypeForeignKey,
			Columns: []string{"COD_EMPRESA"}, RefRelation: "PEDIDO", RefColumns: []string{"COD_EMPRESA"},
		},
		{Name: "CHK_QTD", Relation: "ITEM", Type: constraintTypeCheck, Check: "CHECK (QTD > 0)"},
	}
	ddl := buildConstraintsDDLScript(constraints, "1")
	orphans := buildOrphanRowsScript(constraints, "1")
	for name, script := range map[string]string{constraintsScriptName: ddl, orphanRowsScriptName: orphans} {
		if strings.Contains(script, `"`) {
			t.Fatalf("%s must not use double quotes in dialect 1:\n%s", name, script)
		}
	}
	wantDDL := "ALTER TABLE ITEM ADD CONSTRAINT FK_ITEM_PEDIDO FOREIGN KEY (COD_EMPRESA) REFERENCES PEDIDO (COD_EMPRESA);"
	if !strings.Contains(ddl, wantDDL) {
		t.Fatalf("dialect 1 DDL missing or wrong.\nwant: %s\ngot:\n%s", wantDDL, ddl)
	}
	wantOrphan := "SELECT F.* FROM ITEM F WHERE F.COD_EMPRESA IS NOT NULL AND NOT EXISTS (SELECT 1 FROM PEDIDO P WHERE P.COD_EMPRESA = F.COD_EMPRESA);"
	if !strings.Contains(orphans, wantOrphan) {
		t.Fatalf("dialect 1 orphan query missing or wrong.\nwant: %s\ngot:\n%s", wantOrphan, orphans)
	}
	if quoted := buildConstraintsDDLScript(constraints, "3"); !strings.Contains(quoted, `ALTER TABLE "ITEM"`) {
		t.Fatalf("dialect 3 must keep quoted identifiers:\n%s", quoted)
	}
}

func TestConstraintsExtractionScriptAsksForTheCheckSourceLength(t *testing.T) {
	script := constraintsExtractionScript()
	if !strings.Contains(script, "CHAR_LENGTH(t.RDB$TRIGGER_SOURCE)") {
		t.Fatalf("the query must read the full CHECK source length to detect truncation:\n%s", script)
	}
	if !strings.Contains(script, "VARCHAR("+strconv.Itoa(constraintCheckSourceLimit)+")") {
		t.Fatalf("the CAST limit must match constraintCheckSourceLimit:\n%s", script)
	}
}
