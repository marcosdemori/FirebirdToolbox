//go:build windows

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

var appVersion = "1.2.32"
var appCredit = "Feito por Marcos De Mori Laiola"

// Os BATs originais continuam incorporados como referencia/documentacao da logica
// que originou o utilitario. A v1.2.31 executa gfix/gbak diretamente para manter
// todo o log dentro da interface e permitir combinar versoes de origem/destino.
//
//go:embed scripts/*.bat docs/LEIA-ME.md assets/*.ico
var embeddedFiles embed.FS

const (
	WM_CREATE         = 0x0001
	WM_DESTROY        = 0x0002
	WM_CLOSE          = 0x0010
	WM_DRAWITEM       = 0x002B
	WM_COMMAND        = 0x0111
	WM_SETFONT        = 0x0030
	WM_CTLCOLORSTATIC = 0x0138
	WM_SETICON        = 0x0080

	WS_CAPTION     = 0x00C00000
	WS_SYSMENU     = 0x00080000
	WS_MINIMIZEBOX = 0x00020000
	WS_VISIBLE     = 0x10000000
	WS_CHILD       = 0x40000000
	WS_TABSTOP     = 0x00010000
	WS_VSCROLL     = 0x00200000
	WS_BORDER      = 0x00800000

	WS_EX_CLIENTEDGE = 0x00000200

	ES_AUTOHSCROLL = 0x0080
	ES_PASSWORD    = 0x0020
	ES_READONLY    = 0x0800
	ES_MULTILINE   = 0x0004
	ES_AUTOVSCROLL = 0x0040

	CBS_DROPDOWNLIST = 0x0003
	CBS_HASSTRINGS   = 0x0200

	LBS_NOTIFY    = 0x0001
	LB_ADDSTRING  = 0x0180
	LB_SETCURSEL  = 0x0186
	LB_GETCURSEL  = 0x0188
	LBN_SELCHANGE = 1

	CB_ADDSTRING  = 0x0143
	CB_SETCURSEL  = 0x014E
	CB_GETCURSEL  = 0x0147
	CBN_SELCHANGE = 1

	BN_CLICKED = 0

	EM_SETSEL          = 0x00B1
	EM_REPLACESEL      = 0x00C2
	EM_SCROLLCARET     = 0x00B7
	EM_SETMARGINS      = 0x00D3
	EM_SETPASSWORDCHAR = 0x00CC

	EC_LEFTMARGIN  = 0x0001
	EC_RIGHTMARGIN = 0x0002

	SW_HIDE       = 0
	SW_SHOWNORMAL = 1
	SW_SHOW       = 5

	MB_OK           = 0x00000000
	MB_ICONERROR    = 0x00000010
	MB_ICONQUESTION = 0x00000020
	MB_ICONWARNING  = 0x00000030
	MB_ICONINFO     = 0x00000040
	MB_YESNO        = 0x00000004
	IDYES           = 6

	OFN_OVERWRITEPROMPT = 0x00000002
	OFN_HIDEREADONLY    = 0x00000004
	OFN_NOCHANGEDIR     = 0x00000008
	OFN_PATHMUSTEXIST   = 0x00000800
	OFN_FILEMUSTEXIST   = 0x00001000
	OFN_EXPLORER        = 0x00080000

	DEFAULT_GUI_FONT = 17
	COLOR_WINDOW     = 5

	IDC_ARROW = 32512

	CREATE_NO_WINDOW = 0x08000000

	BS_GROUPBOX    = 0x00000007
	BS_OWNERDRAW   = 0x0000000B
	SS_ICON        = 0x00000003
	SS_CENTERIMAGE = 0x00000200
	SS_ETCHEDHORZ  = 0x00000010
	STM_SETICON    = 0x0170

	ILC_MASK         = 0x00000001
	ILC_COLOR32      = 0x00000020
	BCM_SETIMAGELIST = 0x1602
	BUTTON_ICON_LEFT = 0
	ICON_SMALL       = 0
	ICON_BIG         = 1

	PBS_SMOOTH     = 0x00000001
	PBM_SETPOS     = 0x0402
	PBM_SETRANGE32 = 0x0406
	PBM_SETSTATE   = 0x0410
	PBST_NORMAL    = 0x0001
	PBST_ERROR     = 0x0002
	PBST_PAUSED    = 0x0003

	BIF_RETURNONLYFSDIRS = 0x00000001
	BIF_NEWDIALOGSTYLE   = 0x00000040

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002

	FW_NORMAL   = 400
	FW_SEMIBOLD = 600

	ODS_SELECTED = 0x0001
	ODS_DISABLED = 0x0004
	ODS_FOCUS    = 0x0010

	DT_CENTER       = 0x00000001
	DT_VCENTER      = 0x00000004
	DT_SINGLELINE   = 0x00000020
	DT_END_ELLIPSIS = 0x00008000
	TRANSPARENT     = 1
	NULL_BRUSH      = 5
	DI_NORMAL       = 0x0003
	IMAGE_ICON      = 1
)

const (
	idRoutine        = 1001
	idSourceVersion  = 1002
	idRestoreVersion = 1003
	idSource         = 1004
	idSourceBtn      = 1005
	idDest           = 1006
	idDestBtn        = 1007
	idBackupDir      = 1008
	idBackupDirBtn   = 1009
	idFbSource       = 1010
	idFbSourceBtn    = 1011
	idFbTarget       = 1012
	idFbTargetBtn    = 1013
	idService        = 1014
	idSourcePort     = 1015
	idTargetPort     = 1016
	idUser           = 1017
	idPassword       = 1018
	idPasswordToggle = 1019
	idExecute        = 1020
	idStop           = 1021
	idOpenDest       = 1022
	idHelp           = 1023
	idClearLog       = 1024
	idCopyLog        = 1025
	idLog            = 1026
	idStatus         = 1027
	idDescription    = 1028
	idDetailedLog    = 1029
	idCopyPath       = 1030
	idProgress       = 1031
	idProgressText   = 1032
	idResult         = 1033
	idSkipData       = 1034
	idElapsed        = 1035
	idOpenReport     = 1036
	idOpenDatabase   = 1037

	idHelpCopy         = 4101
	idHelpClose        = 4102
	idHelpNav          = 4103
	idHelpSectionTitle = 4104
	idHelpSectionIntro = 4105

	idConflictKeep1  = 4201
	idConflictKeep2  = 4202
	idConflictExport = 4203
	idConflictCancel = 4204

	idHeaderTitle     = 3100
	idSectionRoutine  = 3101
	idSectionFiles    = 3102
	idSectionFirebird = 3103
	idSectionExecute  = 3104
	idSectionLog      = 3105
)

const (
	lblSource     = 2001
	lblDest       = 2002
	lblBackup     = 2003
	lblFbSource   = 2004
	lblFbTarget   = 2005
	lblService    = 2006
	lblSourcePort = 2007
	lblTargetPort = 2008
	lblSourceVer  = 2009
	lblRestoreVer = 2010
	lblSkipData   = 2011
)

// Medidas centrais da interface. Mantê-las nomeadas evita regressões de
// truncamento em rótulos e no texto de andamento.
const (
	uiFilesLabelWidth     int32 = 150
	uiFilesFieldX         int32 = 182
	uiFirebirdLabelWidth  int32 = 160
	uiFirebirdFieldX      int32 = 192
	uiProgressTextWidth   int32 = 780
	uiElapsedTextWidth    int32 = 744
	uiProgressBarWidth    int32 = 780
	uiSkipEditX           int32 = 190
	uiSkipEditWidth       int32 = 360
	uiOpenReportX         int32 = 560
	uiOpenReportWidth     int32 = 150
	uiLocateDatabaseX     int32 = 720
	uiLocateDatabaseWidth int32 = 164
	uiResultShortcutY     int32 = 582
	uiLogEditHeight       int32 = 116
	uiDescriptionWidth    int32 = 850
	uiSectionBorderLift   int32 = 8

	uiHelpWindowWidth   int32 = 850
	uiHelpWindowHeight  int32 = 700
	uiHelpNavWidth      int32 = 180
	uiHelpContentWidth  int32 = 588
	uiHelpContentHeight int32 = 390
	uiHelpBottomY       int32 = 610
)

type routineKind int

const (
	routineRepair routineKind = iota
	routineRepairAdvanced
	routineRestore
)

type routineDef struct {
	Name        string
	Kind        routineKind
	Description string
}

var routines = []routineDef{
	{"Manutenção / conversão", routineRepair, "Valida, executa sweep, gera backup e, se escolhido, converte/restaura em outra versão."},
	{"Recuperar banco corrompido", routineRepairAdvanced, "Recuperação em cópia para corrupção real: diagnóstico, reparo controlado, backup de resgate, teste de restore e relatório."},
	{"Somente restaurar", routineRestore, "Restaura um arquivo BAK/FBK/GBK diretamente na versao escolhida."},
}

func routineNames(items []routineDef) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

var versions = []string{"1.0", "1.5", "2.0", "2.1", "2.5", "3.0", "4.0", "5.0"}
var restoreVersions = []string{"Nao restaurar", "1.0", "1.5", "2.0", "2.1", "2.5", "3.0", "4.0", "5.0"}

type fbDefaults struct {
	Root    string
	Service string
	Port    string
}

var defaults = map[string]fbDefaults{
	"1.0": {`C:\Program Files\Firebird\Firebird_1_0`, "Autom\u00e1tico", "3050"},
	"1.5": {`C:\Program Files\Firebird\Firebird_1_5`, "Autom\u00e1tico", "3050"},
	"2.0": {`C:\Program Files\Firebird\Firebird_2_0`, "Autom\u00e1tico", "3050"},
	"2.1": {`C:\Program Files\Firebird\Firebird_2_1`, "Autom\u00e1tico", "3050"},
	"2.5": {`C:\Program Files\Firebird\Firebird_2_5`, "Autom\u00e1tico", "3050"},
	"3.0": {`C:\Program Files\Firebird\Firebird_3_0`, "Autom\u00e1tico", "3050"},
	"4.0": {`C:\Program Files\Firebird\Firebird_4_0`, "Autom\u00e1tico", "3050"},
	"5.0": {`C:\Program Files\Firebird\Firebird_5_0`, "Autom\u00e1tico", "3050"},
}

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

type point struct{ x, y int32 }

type msg struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type openFileName struct {
	lStructSize       uint32
	hwndOwner         syscall.Handle
	hInstance         syscall.Handle
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        unsafe.Pointer
	dwReserved        uint32
	flagsEx           uint32
}

type browseInfo struct {
	hwndOwner      syscall.Handle
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

type rect struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type buttonImageList struct {
	himl   uintptr
	margin rect
	uAlign uint32
}

type drawItemStruct struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   syscall.Handle
	HDC        syscall.Handle
	RcItem     rect
	ItemData   uintptr
}

type buttonVisual struct {
	Icon    string
	Primary bool
	Danger  bool
	Accent  bool
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")

	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procShowWindow               = user32.NewProc("ShowWindow")
	procUpdateWindow             = user32.NewProc("UpdateWindow")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessageW         = user32.NewProc("DispatchMessageW")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procSendMessageW             = user32.NewProc("SendMessageW")
	procSetWindowTextW           = user32.NewProc("SetWindowTextW")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLen         = user32.NewProc("GetWindowTextLengthW")
	procEnableWindow             = user32.NewProc("EnableWindow")
	procIsWindowEnabled          = user32.NewProc("IsWindowEnabled")
	procMessageBoxW              = user32.NewProc("MessageBoxW")
	procLoadCursorW              = user32.NewProc("LoadCursorW")
	procDestroyIcon              = user32.NewProc("DestroyIcon")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procLoadImageW               = user32.NewProc("LoadImageW")
	procDrawIconEx               = user32.NewProc("DrawIconEx")
	procDrawTextW                = user32.NewProc("DrawTextW")
	procDrawFocusRect            = user32.NewProc("DrawFocusRect")
	procFillRect                 = user32.NewProc("FillRect")
	procFrameRect                = user32.NewProc("FrameRect")
	procInvalidateRect           = user32.NewProc("InvalidateRect")
	procGetWindowLongW           = user32.NewProc("GetWindowLongW")
	procSetWindowLongW           = user32.NewProc("SetWindowLongW")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procSetTextColor             = gdi32.NewProc("SetTextColor")
	procSetBkColor               = gdi32.NewProc("SetBkColor")
	procSetBkMode                = gdi32.NewProc("SetBkMode")
	procSelectObject             = gdi32.NewProc("SelectObject")
	procCreateSolidBrush         = gdi32.NewProc("CreateSolidBrush")
	procCreateFontW              = gdi32.NewProc("CreateFontW")
	procDeleteObject             = gdi32.NewProc("DeleteObject")

	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	procGetDiskFreeSpaceExW  = kernel32.NewProc("GetDiskFreeSpaceExW")
	procGetOEMCP             = kernel32.NewProc("GetOEMCP")
	procMultiByteToWideChar  = kernel32.NewProc("MultiByteToWideChar")
	procGetStockObject       = gdi32.NewProc("GetStockObject")
	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileNameW     = comdlg32.NewProc("GetSaveFileNameW")
	procSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	procShellExecuteW        = shell32.NewProc("ShellExecuteW")
	procSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
	procCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	procCoUninitialize       = ole32.NewProc("CoUninitialize")
	procInitCommonControls   = comctl32.NewProc("InitCommonControls")
	procImageListCreate      = comctl32.NewProc("ImageList_Create")
	procImageListReplaceIcon = comctl32.NewProc("ImageList_ReplaceIcon")
	procImageListDestroy     = comctl32.NewProc("ImageList_Destroy")

	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
	procMoveMemory       = kernel32.NewProc("RtlMoveMemory")
)

var (
	mainHwnd             syscall.Handle
	helpHwnd             syscall.Handle
	helpContentHwnd      syscall.Handle
	helpNavHwnd          syscall.Handle
	helpSectionTitleHwnd syscall.Handle
	helpSectionIntroHwnd syscall.Handle
	conflictHwnd         syscall.Handle
	controls             = map[int]syscall.Handle{}
	guiFont              syscall.Handle
	titleFont            syscall.Handle
	sectionFont          syscall.Handle
	monoFont             syscall.Handle
	logBrush             syscall.Handle
	staticBrushWhite     syscall.Handle
	resultBrushNeutral   syscall.Handle
	resultBrushSuccess   syscall.Handle
	resultBrushWarning   syscall.Handle
	resultBrushError     syscall.Handle
	appIconSmall         syscall.Handle
	appIconBig           syscall.Handle

	runMu         sync.Mutex
	currentCancel context.CancelFunc
	running       bool

	timerMu            sync.Mutex
	runStartedAt       time.Time
	runFinishedAt      time.Time
	runTimerActive     bool
	runTimerGeneration uint64

	logMu           sync.Mutex
	logText         string
	passwordVisible bool
	detailedLog     bool

	resultMu          sync.Mutex
	lastHeader        databaseHeader
	lastBackup        string
	lastReport        string
	lastDatabase      string
	lastRecoveryState string

	recoveryTechnicalLogMu sync.Mutex
	recoveryTechnicalLog   string

	imageLists []uintptr
	liveIcons  []syscall.Handle
	iconCache  = map[string]syscall.Handle{}

	conflictResolverMu       sync.Mutex
	conflictResolverDecision conflictResolutionDecision
	conflictResolverPath     string

	buttonVisuals = map[int]buttonVisual{
		idSourceBtn:      {Icon: "folder"},
		idDestBtn:        {Icon: "folder"},
		idBackupDirBtn:   {Icon: "folder"},
		idFbSourceBtn:    {Icon: "gear"},
		idFbTargetBtn:    {Icon: "gear"},
		idPasswordToggle: {Icon: "eye"},
		idExecute:        {Icon: "play", Primary: true},
		idStop:           {Icon: "stop", Danger: true},
		idOpenDest:       {Icon: "folder"},
		idCopyPath:       {Icon: "copy"},
		idHelp:           {Icon: "help"},
		idOpenReport:     {Icon: "details", Accent: true},
		idOpenDatabase:   {Icon: "database", Accent: true},
		idDetailedLog:    {Icon: "details"},
		idClearLog:       {Icon: "clear"},
		idCopyLog:        {Icon: "copy"},
		idHelpCopy:       {Icon: "copy"},
		idHelpClose:      {Primary: true},
	}
)

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func u16buf(s string) []uint16 {
	r := utf16.Encode([]rune(s))
	return append(r, 0)
}

func makeFilter(parts ...string) []uint16 {
	s := strings.Join(parts, "\x00") + "\x00\x00"
	return utf16.Encode([]rune(s))
}

func rgb(r, g, b byte) uintptr {
	return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16)
}

func messageBox(title, text string, flags uintptr) int {
	r, _, _ := procMessageBoxW.Call(uintptr(mainHwnd), uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(title))), flags)
	return int(r)
}

func icoImageData(data []byte, requested int32) ([]byte, bool) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return nil, false
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count <= 0 || len(data) < 6+count*16 {
		return nil, false
	}

	bestIndex := -1
	bestScore := int64(1 << 62)
	for i := 0; i < count; i++ {
		entry := 6 + i*16
		width := int32(data[entry])
		height := int32(data[entry+1])
		if width == 0 {
			width = 256
		}
		if height == 0 {
			height = 256
		}
		score := int64(abs32(width-requested) + abs32(height-requested))
		if score < bestScore {
			bestScore = score
			bestIndex = i
		}
	}
	if bestIndex < 0 {
		return nil, false
	}

	entry := 6 + bestIndex*16
	size := int(binary.LittleEndian.Uint32(data[entry+8 : entry+12]))
	offset := int(binary.LittleEndian.Uint32(data[entry+12 : entry+16]))
	if size <= 0 || offset < 0 || offset > len(data) || size > len(data)-offset {
		return nil, false
	}
	return data[offset : offset+size], true
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func loadUIIcon(name string, size int32) syscall.Handle {
	data, err := embeddedFiles.ReadFile("assets/" + name + ".ico")
	if err != nil {
		return 0
	}
	image, ok := icoImageData(data, size)
	if !ok || len(image) == 0 {
		return 0
	}
	h, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&image[0])),
		uintptr(len(image)),
		1,
		0x00030000,
		uintptr(size),
		uintptr(size),
		0,
	)
	return syscall.Handle(h)
}

func cachedUIIcon(name string, size int32) syscall.Handle {
	key := fmt.Sprintf("%s:%d", name, size)
	if h := iconCache[key]; h != 0 {
		return h
	}
	h := loadUIIcon(name, size)
	if h != 0 {
		iconCache[key] = h
		liveIcons = append(liveIcons, h)
	}
	return h
}

func addIcon(name string, x, y, size int32) syscall.Handle {
	hwnd := createControl(0, "STATIC", "", WS_CHILD|WS_VISIBLE|SS_ICON, x, y, size, size, 0)
	icon := cachedUIIcon(name, size)
	if icon != 0 {
		procSendMessageW.Call(uintptr(hwnd), STM_SETICON, uintptr(icon), 0)
	}
	return hwnd
}

func sectionHeaderWidth(id int) int32 {
	// O título permanece compacto para não criar uma faixa branca desnecessária
	// dentro da seção. Desde a v1.2.31 a borda superior fica acima do cabeçalho,
	// portanto estas larguras afetam apenas o acabamento do fundo do texto.
	switch id {
	case idSectionRoutine:
		return 150
	case idSectionFiles:
		return 175
	case idSectionFirebird:
		return 235
	case idSectionExecute:
		return 110
	case idSectionLog:
		return 145
	default:
		return 180
	}
}

func addSectionBox(x, y, w, h int32) syscall.Handle {
	// O BS_GROUPBOX desenha a borda superior alguns pixels abaixo do Y informado.
	// Subir somente a moldura (e aumentar a altura na mesma medida) mantém a borda
	// inferior no lugar, mas deixa o traço superior completamente acima do título.
	// Assim o cabeçalho não precisa apagar nenhum trecho da linha.
	return createControl(0, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_GROUPBOX, x, y-uiSectionBorderLift, w, h+uiSectionBorderLift, 0)
}

func addSectionHeader(id int, name, title string, x, y int32) {
	// O título agora fica integralmente dentro do bloco, abaixo da borda superior
	// contínua. A largura curta serve apenas para manter o fundo restrito ao texto.
	headerY := y + 4
	addIcon(name, x, headerY+1, 18)
	h := createControl(0, "STATIC", title, WS_CHILD|WS_VISIBLE, x+26, headerY, sectionHeaderWidth(id), 22, id)
	if sectionFont != 0 {
		procSendMessageW.Call(uintptr(h), WM_SETFONT, uintptr(sectionFont), 1)
	}
}

func createIconButton(id int, text string, x, y, w, h int32) syscall.Handle {
	return createControl(0, "BUTTON", text, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, x, y, w, h, id)
}

func drawText(hdc syscall.Handle, text string, rc *rect, flags uintptr) {
	chars := utf16.Encode([]rune(text))
	if len(chars) == 0 {
		return
	}
	procDrawTextW.Call(uintptr(hdc), uintptr(unsafe.Pointer(&chars[0])), uintptr(len(chars)), uintptr(unsafe.Pointer(rc)), flags)
}

func drawOwnerButton(dis *drawItemStruct) bool {
	visual, ok := buttonVisuals[int(dis.CtlID)]
	if !ok {
		return false
	}
	disabled := dis.ItemState&ODS_DISABLED != 0
	pressed := dis.ItemState&ODS_SELECTED != 0

	bg := rgb(248, 250, 252)
	border := rgb(203, 213, 225)
	fg := rgb(51, 65, 85)
	if visual.Primary {
		bg = rgb(37, 99, 235)
		border = rgb(29, 78, 216)
		fg = rgb(255, 255, 255)
	} else if visual.Danger {
		bg = rgb(255, 247, 247)
		border = rgb(252, 165, 165)
		fg = rgb(185, 28, 28)
	} else if visual.Accent {
		// Atalhos de resultado ficam discretos enquanto indisponíveis e ganham
		// destaque azul assim que existe um relatório/banco válido para abrir.
		bg = rgb(239, 246, 255)
		border = rgb(96, 165, 250)
		fg = rgb(30, 64, 175)
	}
	if pressed {
		if visual.Primary {
			bg = rgb(29, 78, 216)
		} else if visual.Accent {
			bg = rgb(219, 234, 254)
			border = rgb(59, 130, 246)
		} else {
			bg = rgb(226, 232, 240)
		}
	}
	if disabled {
		bg = rgb(241, 245, 249)
		border = rgb(226, 232, 240)
		fg = rgb(148, 163, 184)
	}

	bgBrush, _, _ := procCreateSolidBrush.Call(bg)
	borderBrush, _, _ := procCreateSolidBrush.Call(border)
	procFillRect.Call(uintptr(dis.HDC), uintptr(unsafe.Pointer(&dis.RcItem)), bgBrush)
	procFrameRect.Call(uintptr(dis.HDC), uintptr(unsafe.Pointer(&dis.RcItem)), borderBrush)
	procDeleteObject.Call(bgBrush)
	procDeleteObject.Call(borderBrush)

	procSetBkMode.Call(uintptr(dis.HDC), TRANSPARENT)
	procSetTextColor.Call(uintptr(dis.HDC), fg)
	buttonFont := guiFont
	if (visual.Primary || visual.Accent) && sectionFont != 0 {
		buttonFont = sectionFont
	}
	oldFont, _, _ := procSelectObject.Call(uintptr(dis.HDC), uintptr(buttonFont))
	defer procSelectObject.Call(uintptr(dis.HDC), oldFont)

	offset := int32(0)
	if pressed {
		offset = 1
	}
	textRect := dis.RcItem
	if icon := cachedUIIcon(visual.Icon, 16); icon != 0 {
		ix := dis.RcItem.left + 10 + offset
		iy := dis.RcItem.top + (dis.RcItem.bottom-dis.RcItem.top-16)/2 + offset
		procDrawIconEx.Call(uintptr(dis.HDC), uintptr(ix), uintptr(iy), uintptr(icon), 16, 16, 0, 0, DI_NORMAL)
		textRect.left += 24
	}
	textRect.left += offset
	textRect.top += offset
	drawText(dis.HDC, getWindowText(dis.HwndItem), &textRect, DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)

	if dis.ItemState&ODS_FOCUS != 0 && !disabled {
		focus := dis.RcItem
		focus.left += 3
		focus.top += 3
		focus.right -= 3
		focus.bottom -= 3
		procDrawFocusRect.Call(uintptr(dis.HDC), uintptr(unsafe.Pointer(&focus)))
	}
	return true
}

func attachButtonIcon(id int, name string) {
	hwnd := controls[id]
	if hwnd == 0 {
		showStartupFailure("nao foi possivel criar a janela principal")
		return
	}
	icon := loadUIIcon(name, 16)
	if icon == 0 {
		return
	}
	himl, _, _ := procImageListCreate.Call(16, 16, ILC_COLOR32|ILC_MASK, 1, 1)
	if himl == 0 {
		procDestroyIcon.Call(uintptr(icon))
		return
	}
	idx, _, _ := procImageListReplaceIcon.Call(himl, ^uintptr(0), uintptr(icon))
	procDestroyIcon.Call(uintptr(icon))
	if int32(idx) < 0 {
		procImageListDestroy.Call(himl)
		return
	}
	bil := buttonImageList{himl: himl, margin: rect{left: 5, top: 1, right: 5, bottom: 1}, uAlign: BUTTON_ICON_LEFT}
	procSendMessageW.Call(uintptr(hwnd), BCM_SETIMAGELIST, 0, uintptr(unsafe.Pointer(&bil)))
	imageLists = append(imageLists, himl)
}

func addSectionIcon(name string, x, y int32) {
	hwnd := createControl(0, "STATIC", "", WS_CHILD|WS_VISIBLE|SS_ICON, x, y, 20, 20, 0)
	icon := loadUIIcon(name, 16)
	if icon == 0 {
		return
	}
	procSendMessageW.Call(uintptr(hwnd), STM_SETICON, uintptr(icon), 0)
	liveIcons = append(liveIcons, icon)
}

func loadAppResourceIcon(hinst syscall.Handle, size int32) syscall.Handle {
	// Carrega o mesmo RT_GROUP_ICON incorporado no executável. O go-winres grava
	// o grupo principal como recurso numérico #1. Usar o recurso do próprio PE
	// mantém Explorer, título da janela e barra de tarefas alinhados.
	h, _, _ := procLoadImageW.Call(
		uintptr(hinst),
		uintptr(1), // MAKEINTRESOURCE(1)
		IMAGE_ICON,
		uintptr(size),
		uintptr(size),
		0,
	)
	if h != 0 {
		return syscall.Handle(h)
	}
	// Fallback para builds de desenvolvimento sem recursos Windows preparados.
	return loadUIIcon("flame", size)
}

func setAppIcons() {
	if mainHwnd == 0 {
		return
	}
	if appIconSmall != 0 {
		procSendMessageW.Call(uintptr(mainHwnd), WM_SETICON, ICON_SMALL, uintptr(appIconSmall))
	}
	if appIconBig != 0 {
		procSendMessageW.Call(uintptr(mainHwnd), WM_SETICON, ICON_BIG, uintptr(appIconBig))
	}
}

func cleanupUIResources() {
	for _, h := range imageLists {
		if h != 0 {
			procImageListDestroy.Call(h)
		}
	}
	imageLists = nil
	for _, h := range liveIcons {
		if h != 0 {
			procDestroyIcon.Call(uintptr(h))
		}
	}
	liveIcons = nil
	iconCache = map[string]syscall.Handle{}
}

func setText(id int, text string) {
	h := controls[id]
	if h != 0 {
		procSetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(u16(text))))
	}
}

func getText(id int) string {
	h := controls[id]
	if h == 0 {
		return ""
	}
	n, _, _ := procGetWindowTextLen.Call(uintptr(h))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func getWindowText(hwnd syscall.Handle) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := procGetWindowTextLen.Call(uintptr(hwnd))
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func enableControl(id int, enabled bool) {
	h := controls[id]
	if h == 0 {
		return
	}
	value := uintptr(0)
	if enabled {
		value = 1
	}
	procEnableWindow.Call(uintptr(h), value)
	// Força o redesenho dos botões owner-draw para que atalhos de resultado
	// mudem imediatamente do estado cinza para o destaque quando habilitados.
	procInvalidateRect.Call(uintptr(h), 0, 1)
}

func createControl(exStyle uint32, className, text string, style uint32, x, y, w, h int32, id int) syscall.Handle {
	hwnd, _, _ := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(u16(className))),
		uintptr(unsafe.Pointer(u16(text))),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(mainHwnd), uintptr(id), 0, 0,
	)
	handle := syscall.Handle(hwnd)
	if id != 0 {
		controls[id] = handle
	}
	if guiFont != 0 {
		procSendMessageW.Call(uintptr(handle), WM_SETFONT, uintptr(guiFont), 1)
	}
	return handle
}

func setControlFont(id int, font syscall.Handle) {
	if h := controls[id]; h != 0 && font != 0 {
		procSendMessageW.Call(uintptr(h), WM_SETFONT, uintptr(font), 1)
	}
}

func comboIndex(id int) int {
	r, _, _ := procSendMessageW.Call(uintptr(controls[id]), CB_GETCURSEL, 0, 0)
	if int(r) < 0 {
		return 0
	}
	return int(r)
}

func setCombo(id, idx int) {
	procSendMessageW.Call(uintptr(controls[id]), CB_SETCURSEL, uintptr(idx), 0)
}

func selectedRoutine() routineDef {
	idx := comboIndex(idRoutine)
	if idx < 0 || idx >= len(routines) {
		return routines[0]
	}
	return routines[idx]
}

func selectedSourceVersion() string {
	idx := comboIndex(idSourceVersion)
	if idx < 0 || idx >= len(versions) {
		return "5.0"
	}
	return versions[idx]
}

func selectedRestoreVersion() string {
	idx := comboIndex(idRestoreVersion)
	if idx <= 0 || idx >= len(restoreVersions) {
		return ""
	}
	return restoreVersions[idx]
}

func versionIndex(version string) int {
	for i, v := range versions {
		if v == version {
			return i
		}
	}
	return 0
}

func restoreVersionIndex(version string) int {
	for i, v := range restoreVersions {
		if v == version {
			return i
		}
	}
	return 0
}

func versionDirName(version string) string {
	return "Firebird_" + strings.ReplaceAll(version, ".", "_")
}

func versionUsesBin(version string) bool {
	switch version {
	case "1.0", "1.5", "2.0", "2.1", "2.5":
		return true
	default:
		return false
	}
}

func standardFirebirdRoots(version string) []string {
	dirName := versionDirName(version)
	bases := []string{
		// ProgramW6432 e essencial quando o executavel universal x86 roda em
		// Windows x64: ProgramFiles pode apontar para Program Files (x86).
		os.Getenv("ProgramW6432"),
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		`C:\Program Files`,
		`C:\Program Files (x86)`,
	}
	seen := map[string]bool{}
	var roots []string
	for _, base := range bases {
		base = strings.TrimSpace(base)
		if base == "" {
			continue
		}
		root := filepath.Join(base, "Firebird", dirName)
		key := strings.ToLower(filepath.Clean(root))
		if seen[key] {
			continue
		}
		seen[key] = true
		roots = append(roots, root)
	}
	if d, ok := defaults[version]; ok {
		key := strings.ToLower(filepath.Clean(d.Root))
		if !seen[key] {
			roots = append([]string{d.Root}, roots...)
		}
	}
	return roots
}

func isKnownDefaultFB(s string) bool {
	sl := strings.ToLower(strings.TrimSpace(s))
	if sl == "" {
		return true
	}
	for _, version := range versions {
		for _, root := range standardFirebirdRoots(version) {
			if strings.EqualFold(filepath.Clean(s), filepath.Clean(root)) {
				return true
			}
		}
	}
	return strings.Contains(sl, `\firebird\firebird_`)
}

var registryValuePathRE = regexp.MustCompile(`(?i)\s+REG_(?:SZ|EXPAND_SZ)\s+(.+)$`)
var firebirdVersionOutputRE = regexp.MustCompile(`(?i)(?:firebird(?:\s+server)?\s+|(?:WI|LI)-V)([1-5]\.\d)`)

type windowsServiceCandidate struct {
	Name      string
	ImagePath string
}

var machineRootsOnce sync.Once
var machineRootsCache []string
var serviceCandidatesOnce sync.Once
var serviceCandidatesCache []windowsServiceCandidate
var probeVersionMu sync.Mutex
var probeVersionCache = map[string]string{}

func uniqueCleanPaths(paths []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, raw := range paths {
		p := strings.Trim(strings.TrimSpace(os.ExpandEnv(raw)), `"`)
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		key := strings.ToLower(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

func parseRegistryFirebirdRoots(output string) []string {
	var roots []string
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		match := registryValuePathRE.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		value := strings.Trim(strings.TrimSpace(match[1]), `"`)
		if value == "" {
			continue
		}
		roots = append(roots, os.ExpandEnv(value))
	}
	return uniqueCleanPaths(roots)
}

func registryFirebirdRoots() []string {
	keys := []string{
		`HKLM\SOFTWARE\Firebird Project\Firebird Server\Instances`,
		// Compatibilidade com instaladores/ambientes legados que usaram a
		// grafia sem espaco em FirebirdProject.
		`HKLM\SOFTWARE\FirebirdProject\Firebird Server\Instances`,
	}
	views := []string{"/reg:64", "/reg:32"}
	var roots []string
	for _, key := range keys {
		for _, view := range views {
			ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
			out, _ := runCommandCapture(ctx, "reg.exe", "query", key, view)
			cancel()
			roots = append(roots, parseRegistryFirebirdRoots(out)...)
		}
	}
	return uniqueCleanPaths(roots)
}

func programFilesFirebirdRoots() []string {
	bases := uniqueCleanPaths([]string{
		os.Getenv("ProgramW6432"),
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		`C:\Program Files`,
		`C:\Program Files (x86)`,
	})
	var roots []string
	for _, base := range bases {
		firebirdBase := filepath.Join(base, "Firebird")
		entries, err := os.ReadDir(firebirdBase)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			root := filepath.Join(firebirdBase, entry.Name())
			if fileExists(filepath.Join(root, "gbak.exe")) || fileExists(filepath.Join(root, "bin", "gbak.exe")) {
				roots = append(roots, root)
			}
		}
	}
	return uniqueCleanPaths(roots)
}

func parseWindowsServiceImagePaths(output string) []windowsServiceCandidate {
	var result []windowsServiceCandidate
	current := ""
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, `HKEY_LOCAL_MACHINE\SYSTEM\CURRENTCONTROLSET\SERVICES\`) {
			current = filepath.Base(strings.ReplaceAll(line, `\`, string(filepath.Separator)))
			continue
		}
		if current == "" || !strings.Contains(upper, "IMAGEPATH") {
			continue
		}
		match := registryValuePathRE.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		result = append(result, windowsServiceCandidate{Name: current, ImagePath: strings.TrimSpace(os.ExpandEnv(match[1]))})
	}
	return result
}

func registryServiceCandidates(ctx context.Context) []windowsServiceCandidate {
	var outCandidates []windowsServiceCandidate
	seen := map[string]bool{}
	for _, view := range []string{"/reg:64", "/reg:32"} {
		out, _ := runCommandCapture(ctx, "reg.exe", "query", `HKLM\SYSTEM\CurrentControlSet\Services`, "/s", "/v", "ImagePath", view)
		for _, c := range parseWindowsServiceImagePaths(out) {
			key := strings.ToLower(c.Name + "|" + c.ImagePath)
			if !seen[key] {
				seen[key] = true
				outCandidates = append(outCandidates, c)
			}
		}
	}
	return outCandidates
}

func machineServiceCandidates() []windowsServiceCandidate {
	serviceCandidatesOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		serviceCandidatesCache = registryServiceCandidates(ctx)
	})
	return append([]windowsServiceCandidate(nil), serviceCandidatesCache...)
}

func executablePathFromImagePath(image string) string {
	image = strings.TrimSpace(os.ExpandEnv(image))
	if image == "" {
		return ""
	}
	if strings.HasPrefix(image, `"`) {
		if end := strings.Index(image[1:], `"`); end >= 0 {
			return image[1 : end+1]
		}
	}
	lower := strings.ToLower(image)
	if idx := strings.Index(lower, ".exe"); idx >= 0 {
		return strings.Trim(strings.TrimSpace(image[:idx+4]), `"`)
	}
	fields := strings.Fields(image)
	if len(fields) > 0 {
		return strings.Trim(fields[0], `"`)
	}
	return ""
}

func firebirdRootFromServiceImage(image string) string {
	exePath := executablePathFromImagePath(image)
	if exePath == "" {
		return ""
	}
	base := strings.ToLower(filepath.Base(exePath))
	if base != "firebird.exe" && base != "fbserver.exe" && base != "fb_inet_server.exe" && base != "fbguard.exe" {
		return ""
	}
	dir := filepath.Dir(exePath)
	if strings.EqualFold(filepath.Base(dir), "bin") {
		dir = filepath.Dir(dir)
	}
	return filepath.Clean(dir)
}

func serviceFirebirdRoots() []string {
	var roots []string
	for _, svc := range machineServiceCandidates() {
		if root := firebirdRootFromServiceImage(svc.ImagePath); root != "" {
			roots = append(roots, root)
		}
	}
	return uniqueCleanPaths(roots)
}

func machineFirebirdRoots() []string {
	machineRootsOnce.Do(func() {
		// Descoberta rapida: Registry oficial do Firebird, Program Files e PATH.
		// A varredura dos servicos fica como fallback em detectFirebird para nao
		// atrasar a abertura do Toolbox em maquinas com muitos servicos.
		var roots []string
		roots = append(roots, registryFirebirdRoots()...)
		roots = append(roots, programFilesFirebirdRoots()...)
		if gbak, err := exec.LookPath("gbak.exe"); err == nil {
			root := filepath.Dir(gbak)
			if strings.EqualFold(filepath.Base(root), "bin") {
				root = filepath.Dir(root)
			}
			roots = append(roots, root)
		}
		machineRootsCache = uniqueCleanPaths(roots)
	})
	return append([]string(nil), machineRootsCache...)
}

func parseFirebirdVersionOutput(output string) string {
	match := firebirdVersionOutputRE.FindStringSubmatch(output)
	if len(match) == 2 {
		return match[1]
	}
	lower := strings.ToLower(output)
	for _, version := range versions {
		if strings.Contains(lower, "firebird "+version) || strings.Contains(lower, "-v"+version) {
			return version
		}
	}
	return ""
}

func versionFromFirebirdPath(root string) string {
	lower := strings.ToLower(strings.ReplaceAll(filepath.Clean(root), "/", `\`))
	for _, version := range versions {
		underscored := strings.ReplaceAll(version, ".", "_")
		if strings.Contains(lower, "firebird_"+underscored) || strings.Contains(lower, "firebird "+version) || strings.Contains(lower, "firebird"+version) {
			return version
		}
	}
	return ""
}

func probeFirebirdVersion(root string) string {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return ""
	}
	key := strings.ToLower(root)
	probeVersionMu.Lock()
	if v, ok := probeVersionCache[key]; ok {
		probeVersionMu.Unlock()
		return v
	}
	probeVersionMu.Unlock()

	var detected string
	for _, version := range versions {
		gbak := firebirdExe(root, version, "gbak.exe")
		if !fileExists(gbak) {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		out, _ := runCommandCapture(ctx, gbak, "-z")
		cancel()
		detected = parseFirebirdVersionOutput(out)
		if detected != "" {
			break
		}
	}
	if detected == "" {
		detected = versionFromFirebirdPath(root)
	}
	probeVersionMu.Lock()
	probeVersionCache[key] = detected
	probeVersionMu.Unlock()
	return detected
}

func detectFirebird(version string) string {
	fallback := defaults[version].Root
	candidates := append([]string{}, standardFirebirdRoots(version)...)
	candidates = append(candidates, machineFirebirdRoots()...)
	candidates = uniqueCleanPaths(candidates)

	// Primeiro confirma a versao pelo proprio gbak. Isso permite encontrar
	// instalacoes em D:\, pastas customizadas e instancias nomeadas sem depender
	// do nome usado no computador de desenvolvimento.
	for _, root := range candidates {
		if !fileExists(firebirdExe(root, version, "gbak.exe")) {
			continue
		}
		if probeFirebirdVersion(root) == version {
			return root
		}
	}
	// Fallback seguro para instalacoes antigas em que gbak -z nao identifica a
	// versao de forma parseavel, mas o diretorio segue a nomenclatura oficial.
	for _, root := range candidates {
		if fileExists(firebirdExe(root, version, "gbak.exe")) && versionFromFirebirdPath(root) == version {
			return root
		}
	}

	// Ultimo fallback: instancia instalada como servico em pasta customizada e
	// sem a chave oficial Instances. Essa busca e deliberadamente tardia.
	for _, root := range serviceFirebirdRoots() {
		if !fileExists(firebirdExe(root, version, "gbak.exe")) {
			continue
		}
		if probeFirebirdVersion(root) == version || versionFromFirebirdPath(root) == version {
			return root
		}
	}
	return fallback
}

var servicePortArgRE = regexp.MustCompile(`(?i)(?:^|\s)-p(?:\s+|=)?(\d{1,5})(?:\s|$)`)

func resolveFirebirdServiceName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.EqualFold(name, "gds_db") {
		return "3050"
	}
	if port, err := net.LookupPort("tcp", name); err == nil && port > 0 && port <= 65535 {
		return strconv.Itoa(port)
	}
	return ""
}

func parseFirebirdConfigPort(data string) string {
	var remotePort string
	var remoteName string
	for _, raw := range strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(strings.SplitN(parts[1], "#", 2)[0])
		switch {
		case strings.EqualFold(key, "RemoteServiceName"):
			remoteName = value
		case strings.EqualFold(key, "RemoteServicePort"):
			if port, err := strconv.Atoi(value); err == nil && port > 0 && port <= 65535 {
				remotePort = strconv.Itoa(port)
			}
		}
	}
	// O Firebird da prioridade ao RemoteServiceName quando ambos estao ativos.
	if resolved := resolveFirebirdServiceName(remoteName); resolved != "" {
		return resolved
	}
	return remotePort
}

func portFromServiceImagePath(image string) string {
	match := servicePortArgRE.FindStringSubmatch(strings.TrimSpace(image))
	if len(match) != 2 {
		return ""
	}
	port, err := strconv.Atoi(match[1])
	if err != nil || port <= 0 || port > 65535 {
		return ""
	}
	return strconv.Itoa(port)
}

func serviceConfiguredPort(root string) string {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return ""
	}
	for _, svc := range machineServiceCandidates() {
		serviceRoot := firebirdRootFromServiceImage(svc.ImagePath)
		if serviceRoot == "" || !strings.EqualFold(filepath.Clean(serviceRoot), root) {
			continue
		}
		if port := portFromServiceImagePath(svc.ImagePath); port != "" {
			return port
		}
	}
	return ""
}

func detectFirebirdPort(root, version string) string {
	fallback := defaults[version].Port
	root = strings.TrimRight(strings.TrimSpace(root), `\/`)
	if root == "" {
		return fallback
	}
	config := filepath.Join(root, "firebird.conf")
	if data, err := os.ReadFile(config); err == nil {
		if port := parseFirebirdConfigPort(string(data)); port != "" {
			return port
		}
	}
	// Instalacoes antigas ou instancias iniciadas com -p podem sobrescrever a
	// configuracao de arquivo. O ImagePath do servico e consultado como fallback.
	if port := serviceConfiguredPort(root); port != "" {
		return port
	}
	return fallback
}

func decodeWindowsBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if utf8.Valid(data) {
		return string(data)
	}
	cp, _, _ := procGetOEMCP.Call()
	if cp == 0 {
		cp = 437
	}
	n, _, _ := procMultiByteToWideChar.Call(cp, 0, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 0, 0)
	if n == 0 {
		return strings.ToValidUTF8(string(data), "?")
	}
	buf := make([]uint16, int(n))
	procMultiByteToWideChar.Call(cp, 0, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(unsafe.Pointer(&buf[0])), n)
	return string(utf16.Decode(buf))
}

func runCommandCapture(ctx context.Context, exe string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(decodeWindowsBytes(out.Bytes())), err
}

func portReachable(port string) bool {
	port = strings.TrimSpace(port)
	if port == "" {
		return false
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), 1200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func serviceLooksAutomatic(service string) bool {
	service = strings.ToLower(strings.TrimSpace(service))
	return service == "" || service == "-" || strings.HasPrefix(service, "autom")
}

func serviceQueryOK(ctx context.Context, name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	_, err := runCommandCapture(ctx, "sc.exe", "query", name)
	return err == nil
}

func detectFirebirdService(ctx context.Context, root, preferred string) string {
	if !serviceLooksAutomatic(preferred) && serviceQueryOK(ctx, preferred) {
		return preferred
	}
	rootLower := strings.ToLower(filepath.Clean(strings.TrimSpace(root)))
	fallback := ""

	// Caminho rapido: nomes padrao do instalador normalmente contem Firebird.
	output, _ := runCommandCapture(ctx, "sc.exe", "query", "type=", "service", "state=", "all")
	seen := map[string]bool{}
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.Contains(strings.ToLower(line), "firebird") || !strings.Contains(line, ":") {
			continue
		}
		name := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		if name == "" || seen[strings.ToLower(name)] || !serviceQueryOK(ctx, name) {
			continue
		}
		seen[strings.ToLower(name)] = true
		if fallback == "" {
			fallback = name
		}
		qc, _ := runCommandCapture(ctx, "sc.exe", "qc", name)
		if rootLower != "" && strings.Contains(strings.ToLower(qc), rootLower) {
			return name
		}
	}

	// Fallback portavel: consulta ImagePath dos servicos no Registry. Assim uma
	// instancia renomeada continua sendo encontrada mesmo que o nome do servico
	// nao contenha "Firebird".
	for _, svc := range machineServiceCandidates() {
		serviceRoot := firebirdRootFromServiceImage(svc.ImagePath)
		if serviceRoot == "" || !serviceQueryOK(ctx, svc.Name) {
			continue
		}
		if fallback == "" {
			fallback = svc.Name
		}
		if rootLower != "" && strings.EqualFold(filepath.Clean(serviceRoot), filepath.Clean(root)) {
			return svc.Name
		}
	}
	return fallback
}

func verifyFirebirdEndpoint(ctx context.Context, label, version, root, service, port string) (string, error) {
	appendLog("Verificando Firebird " + label + "...")
	resolvedPort := strings.TrimSpace(port)
	if !portReachable(resolvedPort) {
		configured := detectFirebirdPort(root, version)
		if configured != resolvedPort && portReachable(configured) {
			resolvedPort = configured
			appendLog("Porta detectada automaticamente: " + resolvedPort + ".")
		} else {
			return "", fmt.Errorf("não foi possível conectar ao Firebird %s em 127.0.0.1:%s", version, resolvedPort)
		}
	}
	appendLog("\u2713 Firebird dispon\u00edvel em 127.0.0.1:" + resolvedPort + ".")
	if svc := detectFirebirdService(ctx, root, service); svc != "" {
		appendLog("Servi\u00e7o detectado: " + svc + ".")
	} else if detailedLog {
		appendLog("Servi\u00e7o Windows n\u00e3o identificado; a porta est\u00e1 respondendo normalmente.")
	}
	return resolvedPort, nil
}

func firebirdExe(root, version, exe string) string {
	root = strings.TrimRight(strings.TrimSpace(root), `\/`)
	if root == "" {
		return ""
	}
	if strings.EqualFold(filepath.Base(root), "bin") {
		candidate := filepath.Join(root, exe)
		if fileExists(candidate) || versionUsesBin(version) {
			return candidate
		}
		root = filepath.Dir(root)
	}

	rootCandidate := filepath.Join(root, exe)
	binCandidate := filepath.Join(root, "bin", exe)
	if fileExists(rootCandidate) {
		return rootCandidate
	}
	if fileExists(binCandidate) {
		return binCandidate
	}
	if versionUsesBin(version) {
		return binCandidate
	}
	return rootCandidate
}

func normalizeFBRoot(selectedExe, version string) string {
	dir := filepath.Dir(selectedExe)
	if strings.EqualFold(filepath.Base(dir), "bin") {
		return filepath.Dir(dir)
	}
	return dir
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func suffixForVersion(version string) string {
	switch version {
	case "1.0":
		return "10.RED"
	case "1.5":
		return "15.RED"
	case "2.0":
		return "20.RED"
	case "2.1":
		return "21.RED"
	case "2.5":
		return "25.RED"
	case "3.0":
		return "30.RED"
	case "4.0":
		return "40.RED"
	case "5.0":
		return "50.RED"
	default:
		return ".RED"
	}
}

func suggestDestination(src, version string) string {
	if src == "" || version == "" {
		return ""
	}
	dir := filepath.Dir(src)
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	for _, suffix := range []string{"10", "15", "20", "21", "25", "30", "40", "50"} {
		base = strings.TrimSuffix(base, suffix)
	}
	base = strings.TrimSpace(strings.TrimSuffix(base, "-"))
	if base == "" {
		base = "DBCOM"
	}
	return filepath.Join(dir, base+suffixForVersion(version))
}

func openDialog(title string, backup bool) string {
	buf := make([]uint16, 32768)
	var filter []uint16
	if backup {
		filter = makeFilter("Backups Firebird (*.BAK;*.FBK;*.GBK)", "*.BAK;*.FBK;*.GBK", "Todos os arquivos (*.*)", "*.*")
	} else {
		filter = makeFilter("Banco Firebird (*.FDB;*.RED)", "*.FDB;*.RED", "Todos os arquivos (*.*)", "*.*")
	}
	t := u16buf(title)
	ofn := openFileName{
		lStructSize: uint32(unsafe.Sizeof(openFileName{})),
		hwndOwner:   mainHwnd,
		lpstrFilter: &filter[0],
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		lpstrTitle:  &t[0],
		flags:       OFN_EXPLORER | OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR | OFN_HIDEREADONLY,
	}
	r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func saveDialog(title, initial string) string {
	buf := make([]uint16, 32768)
	copy(buf, u16buf(initial))
	filter := makeFilter("Banco Firebird (*.RED;*.FDB)", "*.RED;*.FDB", "Todos os arquivos (*.*)", "*.*")
	t := u16buf(title)
	def := u16buf("RED")
	ofn := openFileName{
		lStructSize: uint32(unsafe.Sizeof(openFileName{})),
		hwndOwner:   mainHwnd,
		lpstrFilter: &filter[0],
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		lpstrTitle:  &t[0],
		lpstrDefExt: &def[0],
		flags:       OFN_EXPLORER | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR | OFN_OVERWRITEPROMPT,
	}
	r, _, _ := procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func chooseFolder(title string) string {
	display := make([]uint16, 260)
	t := u16buf(title)
	bi := browseInfo{
		hwndOwner:      mainHwnd,
		pszDisplayName: &display[0],
		lpszTitle:      &t[0],
		ulFlags:        BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE,
	}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer procCoTaskMemFree.Call(pidl)
	pathBuf := make([]uint16, 32768)
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(pathBuf)
}

func chooseGbak(version string) string {
	buf := make([]uint16, 32768)
	filter := makeFilter("gbak.exe", "gbak.exe", "Executaveis (*.exe)", "*.exe", "Todos os arquivos (*.*)", "*.*")
	title := u16buf("Selecione o gbak.exe do Firebird " + version)
	ofn := openFileName{
		lStructSize: uint32(unsafe.Sizeof(openFileName{})),
		hwndOwner:   mainHwnd,
		lpstrFilter: &filter[0],
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		lpstrTitle:  &title[0],
		flags:       OFN_EXPLORER | OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR,
	}
	r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func appendLog(text string) {
	text = strings.TrimRight(text, "\r\n")
	if text == "" {
		return
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	stamp := time.Now().Format("15:04:05")
	var b strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString("[")
		b.WriteString(stamp)
		b.WriteString("] ")
		b.WriteString(line)
		b.WriteString("\r\n")
	}
	chunk := b.String()
	if chunk == "" {
		return
	}

	logMu.Lock()
	logText += chunk
	if len(logText) > 250000 {
		logText = logText[len(logText)-200000:]
	}
	logMu.Unlock()

	h := controls[idLog]
	if h == 0 {
		return
	}
	procSendMessageW.Call(uintptr(h), EM_SETSEL, ^uintptr(0), ^uintptr(0))
	procSendMessageW.Call(uintptr(h), EM_REPLACESEL, 0, uintptr(unsafe.Pointer(u16(chunk))))
	procSendMessageW.Call(uintptr(h), EM_SCROLLCARET, 0, 0)
}

func clearLog() {
	logMu.Lock()
	logText = ""
	logMu.Unlock()
	setText(idLog, "")
}

func copyTextToClipboard(text, successStatus string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	if r, _, _ := procOpenClipboard.Call(uintptr(mainHwnd)); r == 0 {
		messageBox("Firebird Toolbox", "Não foi possível abrir a área de transferência.", MB_OK|MB_ICONWARNING)
		return false
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()

	data := u16buf(text)
	size := uintptr(len(data) * 2)
	hMem, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, size)
	if hMem == 0 {
		return false
	}
	p, _, _ := procGlobalLock.Call(hMem)
	if p == 0 {
		procGlobalFree.Call(hMem)
		return false
	}
	procMoveMemory.Call(p, uintptr(unsafe.Pointer(&data[0])), size)
	procGlobalUnlock.Call(hMem)
	if r, _, _ := procSetClipboardData.Call(CF_UNICODETEXT, hMem); r == 0 {
		procGlobalFree.Call(hMem)
		return false
	}
	if successStatus != "" {
		setText(idStatus, successStatus)
	}
	return true
}

func progressPhaseText(percent int, text string) string {
	if text == "" || !isRunning() {
		return text
	}
	r := selectedRoutine()
	target := selectedRestoreVersion()
	phase, total := 1, 1
	switch r.Kind {
	case routineRepairAdvanced:
		total = 6
		switch {
		case percent < 15:
			phase = 1
		case percent < 30:
			phase = 2
		case percent < 55:
			phase = 3
		case percent < 86:
			phase = 4
		case percent < 94:
			phase = 5
		default:
			phase = 6
		}
	case routineRepair:
		total = 3
		if target != "" {
			total = 4
		}
		switch {
		case percent < 30:
			phase = 1
		case percent < 48:
			phase = 2
		case percent < 68 || target == "":
			phase = 3
		default:
			phase = 4
		}
	case routineRestore:
		total = 2
		if percent >= 90 {
			phase = 2
		}
	}
	return fmt.Sprintf("Fase %d/%d • %s", phase, total, text)
}

func setProgress(percent int, text string) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	if h := controls[idProgress]; h != 0 {
		procSendMessageW.Call(uintptr(h), PBM_SETPOS, uintptr(percent), 0)
	}
	if text != "" {
		setText(idProgressText, progressPhaseText(percent, text))
	}
}

func setProgressState(state uintptr) {
	if h := controls[idProgress]; h != 0 {
		procSendMessageW.Call(uintptr(h), PBM_SETSTATE, state, 0)
	}
}

func formatElapsedDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int64(d / time.Second)
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d min %02d s", minutes, seconds)
}

func elapsedDurationSnapshot() time.Duration {
	timerMu.Lock()
	defer timerMu.Unlock()
	if runStartedAt.IsZero() {
		return 0
	}
	if runTimerActive {
		return time.Since(runStartedAt)
	}
	if runFinishedAt.IsZero() {
		return 0
	}
	return runFinishedAt.Sub(runStartedAt)
}

func elapsedTextSnapshot() string {
	return formatElapsedDuration(elapsedDurationSnapshot())
}

func startRunTimer() {
	timerMu.Lock()
	runStartedAt = time.Now()
	runFinishedAt = time.Time{}
	runTimerActive = true
	runTimerGeneration++
	generation := runTimerGeneration
	timerMu.Unlock()

	setText(idElapsed, formatElapsedDuration(0))

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			timerMu.Lock()
			if !runTimerActive || generation != runTimerGeneration {
				timerMu.Unlock()
				return
			}
			elapsed := time.Since(runStartedAt)
			timerMu.Unlock()
			setText(idElapsed, formatElapsedDuration(elapsed))
		}
	}()
}

func finishRunTimer() string {
	timerMu.Lock()
	if runStartedAt.IsZero() {
		timerMu.Unlock()
		setText(idElapsed, formatElapsedDuration(0))
		return formatElapsedDuration(0)
	}
	if runTimerActive {
		runFinishedAt = time.Now()
		runTimerActive = false
		runTimerGeneration++
	}
	finished := runFinishedAt
	started := runStartedAt
	timerMu.Unlock()

	if finished.IsZero() {
		finished = time.Now()
	}
	text := formatElapsedDuration(finished.Sub(started))
	setText(idElapsed, text)
	return text
}

// setTerminalProgress encerra visualmente a barra em 100% para qualquer estado
// terminal do fluxo. O ProgressBar nativo do Windows anima PBM_SETPOS; a pequena
// oscilacao 100 -> 99 -> 100 elimina o atraso visual que podia deixar a barra
// aparentemente incompleta quando a dialog final era aberta imediatamente.
func setTerminalProgress(state uintptr, text string) {
	setProgressState(state)
	if h := controls[idProgress]; h != 0 {
		procSendMessageW.Call(uintptr(h), PBM_SETPOS, 100, 0)
		procSendMessageW.Call(uintptr(h), PBM_SETPOS, 99, 0)
		procSendMessageW.Call(uintptr(h), PBM_SETPOS, 100, 0)
		procInvalidateRect.Call(uintptr(h), 0, 1)
		procUpdateWindow.Call(uintptr(h))
	}
	if text != "" {
		setText(idProgressText, text)
	}
	if mainHwnd != 0 {
		procUpdateWindow.Call(uintptr(mainHwnd))
	}
}

func resetRunResult() {
	resultMu.Lock()
	lastHeader = databaseHeader{}
	lastBackup = ""
	lastReport = ""
	lastDatabase = ""
	lastRecoveryState = ""
	resultMu.Unlock()
	setText(idResult, "Aguardando execução")
	setText(idElapsed, formatElapsedDuration(0))
	enableControl(idOpenReport, false)
	enableControl(idOpenDatabase, false)
	setProgressState(PBST_NORMAL)
	setProgress(0, "Preparando...")
}

func setLastBackup(path string) {
	resultMu.Lock()
	lastBackup = path
	resultMu.Unlock()
}

func setLastHeader(h databaseHeader) {
	resultMu.Lock()
	lastHeader = h
	resultMu.Unlock()
}

func setLastReport(path string) {
	resultMu.Lock()
	lastReport = path
	resultMu.Unlock()
	if fileExists(path) && !isRunning() {
		enableControl(idOpenReport, true)
	}
}

func setLastDatabase(path string) {
	resultMu.Lock()
	lastDatabase = path
	resultMu.Unlock()
}

func lastDatabaseSnapshot() string {
	resultMu.Lock()
	defer resultMu.Unlock()
	return lastDatabase
}

func fileGeneratedThisRun(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	timerMu.Lock()
	started := runStartedAt
	timerMu.Unlock()
	if started.IsZero() {
		return false
	}
	// Sistemas de arquivos do Windows podem arredondar timestamps; a pequena
	// tolerância evita falso negativo sem aceitar um arquivo antigo qualquer.
	return !info.ModTime().Before(started.Add(-2 * time.Second))
}

func setLastRecoveryState(state string) {
	resultMu.Lock()
	lastRecoveryState = state
	resultMu.Unlock()
}

func recoveryStateSnapshot() string {
	resultMu.Lock()
	defer resultMu.Unlock()
	return lastRecoveryState
}

func runResultSnapshot() (databaseHeader, string, string) {
	resultMu.Lock()
	defer resultMu.Unlock()
	return lastHeader, lastBackup, lastReport
}

func successSummary(c workflowConfig) (string, string) {
	h, backup, report := runResultSnapshot()
	recoveryState := recoveryStateSnapshot()
	upperState := strings.ToUpper(recoveryState)
	partial := strings.Contains(upperState, "PARCIAL")
	compact := "✓ Concluído"
	if partial {
		compact = "⚠ Recuperação parcial"
	}
	if c.TargetVersion != "" {
		compact += " | FB " + c.TargetVersion
	}
	if h.ODS != "" {
		compact += " | ODS " + h.ODS
	}
	if h.Dialect != "" {
		compact += " | Dialeto " + h.Dialect
	}

	lines := []string{"RESUMO FINAL"}
	if partial {
		lines = append(lines, "⚠ Recuperação parcial concluída")
	} else {
		lines = append(lines, "✓ Rotina concluída com sucesso")
	}
	lines = append(lines, "", "Rotina: "+c.Routine.Name)
	// a versao de destino identifica para qual Firebird o banco foi convertido
	if c.TargetVersion != "" {
		lines = append(lines, "Firebird de destino: "+c.TargetVersion)
	}
	if c.Routine.Kind == routineRepairAdvanced {
		lines = append(lines, "✓ Banco original preservado")
	}
	if backup != "" && fileExists(backup) {
		lines = append(lines, "✓ Backup criado e preservado")
	}
	if c.Dest != "" && fileExists(c.Dest) {
		lines = append(lines, "✓ Banco de destino criado: "+c.Dest)
	}
	if report != "" && fileExists(report) {
		lines = append(lines, "✓ Relatório disponível: "+report)
	}
	if h.ODS != "" || h.Dialect != "" {
		validation := "✓ Resultado validado"
		if h.ODS != "" {
			validation += " | ODS " + h.ODS
		}
		if h.Dialect != "" {
			validation += " | Dialeto " + h.Dialect
		}
		lines = append(lines, validation)
	}
	if recoveryState != "" {
		lines = append(lines, "Estado: "+recoveryState)
	}
	lines = append(lines, "Tempo total: "+elapsedTextSnapshot())
	if c.Routine.Kind == routineRepairAdvanced && strings.Contains(upperState, "APTO PARA PRODUÇÃO") && c.Dest != "" {
		lines = append(lines, "", "IMPORTANTE: o Toolbox não substitui automaticamente o banco usado pelo sistema.", "Confirme que a aplicação aponta para o banco de destino aprovado antes de reabrir o sistema.")
	}
	lines = append(lines, "", "Use os botões Localizar banco, Abrir relatório e Abrir pasta para acessar os resultados.")
	return strings.Join(lines, "\n"), compact
}

func failureSummary(err error) string {
	state := recoveryStateSnapshot()
	upperState := strings.ToUpper(state)
	message := "A rotina foi encerrada com erro."
	if strings.Contains(upperState, "NÃO APTO PARA PRODUÇÃO") {
		message = "O banco foi recuperado/restaurado, mas reprovou um ou mais gates críticos de saúde e NÃO deve ser colocado em produção."
	} else if strings.Contains(upperState, "AGUARDANDO DECISÃO") {
		message = "A recuperação foi interrompida para preservar um conflito de dados que exige decisão manual."
	} else if strings.Contains(upperState, "PARCIAL") {
		message = "A recuperação produziu apenas um resultado parcial para inspeção; o banco não foi aprovado como recuperação completa."
	}
	if err != nil {
		message += "\n\n" + err.Error()
	}
	_, _, report := runResultSnapshot()
	message += "\n\nTempo total: " + elapsedTextSnapshot()
	if state != "" {
		message += "\n\nEstado da recuperação: " + state
	}
	if report != "" {
		message += "\n\nRelatório de recuperação: " + report
		if strings.Contains(upperState, "NÃO APTO PARA PRODUÇÃO") {
			healthPath := filepath.Join(filepath.Dir(report), "RELATORIO_SAUDE_PERFORMANCE.txt")
			if fileExists(healthPath) {
				message += "\nRelatório de saúde: " + healthPath
			}
		}
	}
	return message + "\n\nConsulte o Log de execução para os detalhes técnicos."
}

func copyLogToClipboard() {
	logMu.Lock()
	text := logText
	logMu.Unlock()
	copyTextToClipboard(text, "Log copiado para a área de transferência.")
}

func copyDestinationPath() {
	path := ""
	r := selectedRoutine()
	if selectedRestoreVersion() != "" || r.Kind == routineRestore {
		path = strings.TrimSpace(getText(idDest))
	}
	if path == "" {
		_, backup, _ := runResultSnapshot()
		path = backup
	}
	if path == "" {
		path = strings.TrimSpace(getText(idBackupDir))
	}
	if path == "" {
		messageBox("Firebird Toolbox", "Ainda não há um caminho de resultado para copiar.", MB_OK|MB_ICONINFO)
		return
	}
	copyTextToClipboard(path, "Caminho do resultado copiado.")
}

func setRunning(value bool) {
	runMu.Lock()
	running = value
	runMu.Unlock()
	enableControl(idExecute, !value)
	enableControl(idStop, value)
	for _, id := range []int{idRoutine, idSourceVersion, idRestoreVersion, idSource, idSourceBtn, idDest, idDestBtn, idBackupDir, idBackupDirBtn, idFbSource, idFbSourceBtn, idFbTarget, idFbTargetBtn, idService, idSourcePort, idTargetPort, idUser, idPassword, idPasswordToggle, idDetailedLog, idSkipData} {
		if value {
			enableControl(id, false)
		}
	}
	if !value {
		updateUIState(false)
	}
}

func isRunning() bool {
	runMu.Lock()
	defer runMu.Unlock()
	return running
}

func sourceVersionChanged() {
	version := selectedSourceVersion()
	if isKnownDefaultFB(getText(idFbSource)) {
		setText(idFbSource, detectFirebird(version))
	}
	d := defaults[version]
	setText(idService, d.Service)
	setText(idSourcePort, detectFirebirdPort(getText(idFbSource), version))

	r := selectedRoutine()
	if r.Kind == routineRepair || r.Kind == routineRepairAdvanced {
		target := selectedRestoreVersion()
		if target == "" {
			setCombo(idRestoreVersion, restoreVersionIndex(version))
			target = version
		}
		if target == version && isKnownDefaultFB(getText(idFbTarget)) {
			setText(idFbTarget, detectFirebird(target))
			setText(idTargetPort, detectFirebirdPort(getText(idFbTarget), target))
		}
		if src := strings.TrimSpace(getText(idSource)); src != "" && target != "" {
			setText(idDest, suggestDestination(src, target))
		}
	}
	updateUIState(false)
}

func restoreVersionChanged() {
	target := selectedRestoreVersion()
	if target == "" {
		setText(idFbTarget, "")
		setText(idTargetPort, "")
		enableControl(idDest, false)
		enableControl(idDestBtn, false)
		enableControl(idFbTarget, false)
		enableControl(idFbTargetBtn, false)
		enableControl(idTargetPort, false)
		return
	}
	if isKnownDefaultFB(getText(idFbTarget)) || strings.TrimSpace(getText(idFbTarget)) == "" {
		setText(idFbTarget, detectFirebird(target))
	}
	setText(idTargetPort, detectFirebirdPort(getText(idFbTarget), target))
	if src := strings.TrimSpace(getText(idSource)); src != "" {
		setText(idDest, suggestDestination(src, target))
	}
	updateUIState(false)
}

func routineChanged() {
	r := selectedRoutine()
	switch r.Kind {
	case routineRepair:
		if selectedRestoreVersion() == "" {
			setCombo(idRestoreVersion, restoreVersionIndex(selectedSourceVersion()))
		}
	case routineRepairAdvanced:
		if selectedRestoreVersion() == "" {
			setCombo(idRestoreVersion, restoreVersionIndex(selectedSourceVersion()))
		}
	case routineRestore:
		if selectedRestoreVersion() == "" {
			setCombo(idRestoreVersion, restoreVersionIndex("5.0"))
		}
	}
	setText(idDescription, r.Description)
	sourceVersionChanged()
	restoreVersionChanged()
	updateUIState(false)
}

func updateUIState(forceDisabled bool) {
	if forceDisabled || isRunning() {
		return
	}
	r := selectedRoutine()
	target := selectedRestoreVersion()

	sourceVersionEnabled := r.Kind == routineRepair || r.Kind == routineRepairAdvanced
	restoreVersionEnabled := r.Kind == routineRepair || r.Kind == routineRepairAdvanced || r.Kind == routineRestore
	sourceFBEnabled := r.Kind != routineRestore
	backupDirEnabled := r.Kind == routineRepair || r.Kind == routineRepairAdvanced
	destEnabled := target != ""
	targetFBEnabled := destEnabled

	enableControl(idSourceVersion, sourceVersionEnabled)
	enableControl(idRestoreVersion, restoreVersionEnabled)
	enableControl(idFbSource, sourceFBEnabled)
	enableControl(idFbSourceBtn, sourceFBEnabled)
	enableControl(idService, sourceFBEnabled)
	enableControl(idSourcePort, sourceFBEnabled)
	enableControl(idBackupDir, backupDirEnabled)
	enableControl(idBackupDirBtn, backupDirEnabled)
	enableControl(idDest, destEnabled)
	enableControl(idDestBtn, destEnabled)
	enableControl(idFbTarget, targetFBEnabled)
	enableControl(idFbTargetBtn, targetFBEnabled)
	enableControl(idTargetPort, targetFBEnabled)
	enableControl(idUser, true)
	enableControl(idPassword, true)
	enableControl(idPasswordToggle, true)
	enableControl(idSource, true)
	enableControl(idSourceBtn, true)
	advancedSkipEnabled := r.Kind == routineRepairAdvanced && versionNumber(selectedSourceVersion()) >= 3.0
	enableControl(idSkipData, advancedSkipEnabled)
	enableControl(lblSkipData, r.Kind == routineRepairAdvanced)
	if r.Kind == routineRepairAdvanced {
		if advancedSkipEnabled {
			setText(lblSkipData, "Pular tabelas (regex):")
		} else {
			setText(lblSkipData, "Pular tabelas: FB 3.0+")
		}
	} else {
		setText(lblSkipData, "Pular tabelas (avançado):")
	}

	if r.Kind == routineRestore {
		setText(lblSource, "Arquivo de backup:")
	} else {
		setText(lblSource, "Banco de origem:")
	}

	if r.Kind == routineRestore {
		setText(lblFbSource, "Firebird da rotina:")
		setText(idFbSource, "Não utilizado nesta rotina")
		setText(idService, "-")
		setText(idSourcePort, "-")
	} else if strings.TrimSpace(getText(idFbSource)) == "Não utilizado nesta rotina" {
		version := selectedSourceVersion()
		setText(idFbSource, detectFirebird(version))
		setText(idService, defaults[version].Service)
		setText(idSourcePort, defaults[version].Port)
	}

	if !backupDirEnabled {
		if r.Kind == routineRestore {
			setText(idBackupDir, "Não utilizado na restauração simples")
		}
	} else if strings.HasPrefix(getText(idBackupDir), "Não utilizado") {
		if src := strings.TrimSpace(getText(idSource)); src != "" {
			setText(idBackupDir, filepath.Dir(src))
		} else {
			setText(idBackupDir, "")
		}
	}
}

func browseSource() {
	r := selectedRoutine()
	backup := r.Kind == routineRestore
	p := openDialog("Selecione o arquivo de origem", backup)
	if p == "" {
		return
	}
	setText(idSource, p)
	if r.Kind == routineRepair || r.Kind == routineRepairAdvanced {
		setText(idBackupDir, filepath.Dir(p))
	}
	if target := selectedRestoreVersion(); target != "" {
		setText(idDest, suggestDestination(p, target))
	}
}

func browseDestination() {
	target := selectedRestoreVersion()
	if target == "" {
		return
	}
	initial := strings.TrimSpace(getText(idDest))
	if initial == "" {
		initial = suggestDestination(strings.TrimSpace(getText(idSource)), target)
	}
	p := saveDialog("Escolha o arquivo de destino", initial)
	if p != "" {
		setText(idDest, p)
	}
}

func browseBackupDir() {
	p := chooseFolder("Selecione a pasta onde o backup será salvo")
	if p != "" {
		setText(idBackupDir, p)
	}
}

func browseFB(target bool) {
	version := selectedSourceVersion()
	id := idFbSource
	if target {
		version = selectedRestoreVersion()
		id = idFbTarget
	}
	if version == "" {
		return
	}
	p := chooseGbak(version)
	if p == "" {
		return
	}
	root := normalizeFBRoot(p, version)
	setText(id, root)
	if target {
		setText(idTargetPort, detectFirebirdPort(root, version))
	} else {
		setText(idSourcePort, detectFirebirdPort(root, version))
	}
}

func passwordEditStyle(style uint32, visible bool) uint32 {
	if visible {
		return style &^ ES_PASSWORD
	}
	return style | ES_PASSWORD
}

func applyPasswordVisibility(visible bool) {
	h := controls[idPassword]
	if h == 0 {
		return
	}
	idx := int32(-16) // GWL_STYLE
	styleRaw, _, _ := procGetWindowLongW.Call(uintptr(h), uintptr(uint32(idx)))
	style := passwordEditStyle(uint32(styleRaw), visible)
	procSetWindowLongW.Call(uintptr(h), uintptr(uint32(idx)), uintptr(style))

	mask := uintptr('*')
	if visible {
		mask = 0
	}
	procSendMessageW.Call(uintptr(h), EM_SETPASSWORDCHAR, mask, 0)
	procInvalidateRect.Call(uintptr(h), 0, 1)
	procUpdateWindow.Call(uintptr(h))
}

func togglePassword() {
	passwordVisible = !passwordVisible
	if passwordVisible {
		setText(idPasswordToggle, "Ocultar senha")
	} else {
		setText(idPasswordToggle, "Mostrar senha")
	}
	applyPasswordVisibility(passwordVisible)
}

func versionNumber(version string) float64 {
	v, err := strconv.ParseFloat(version, 64)
	if err != nil {
		return 0
	}
	return v
}

func isBackwardMigration(sourceVersion, targetVersion string) bool {
	if sourceVersion == "" || targetVersion == "" {
		return false
	}
	return versionNumber(targetVersion) < versionNumber(sourceVersion)
}

func supportsServiceManagerRestore(version string) bool {
	// Firebird 1.0 fica em modo local por compatibilidade maxima com instalacoes muito antigas.
	return version != "1.0"
}

type workflowConfig struct {
	Routine       routineDef
	SourceVersion string
	TargetVersion string
	Source        string
	Dest          string
	BackupDir     string
	FbSource      string
	FbTarget      string
	Service       string
	SourcePort    string
	TargetPort    string
	User          string
	Password      string
	SkipData      string
}

func collectConfig() workflowConfig {
	return workflowConfig{
		Routine:       selectedRoutine(),
		SourceVersion: selectedSourceVersion(),
		TargetVersion: selectedRestoreVersion(),
		Source:        strings.TrimSpace(getText(idSource)),
		Dest:          strings.TrimSpace(getText(idDest)),
		BackupDir:     strings.TrimSpace(getText(idBackupDir)),
		FbSource:      strings.TrimSpace(getText(idFbSource)),
		FbTarget:      strings.TrimSpace(getText(idFbTarget)),
		Service:       strings.TrimSpace(getText(idService)),
		SourcePort:    strings.TrimSpace(getText(idSourcePort)),
		TargetPort:    strings.TrimSpace(getText(idTargetPort)),
		User:          strings.TrimSpace(getText(idUser)),
		Password:      getText(idPassword),
		SkipData:      strings.TrimSpace(getText(idSkipData)),
	}
}

func validateConfig(c workflowConfig) error {
	if c.Source == "" || !fileExists(c.Source) {
		return errors.New("selecione um arquivo de origem valido")
	}
	if c.User == "" {
		return errors.New("informe o usuário do Firebird")
	}
	if strings.ContainsAny(c.User+c.Password, "\r\n") {
		return errors.New("usuário ou senha contém quebra de linha inválida")
	}

	if c.Routine.Kind != routineRestore {
		if c.FbSource == "" || !fileExists(firebirdExe(c.FbSource, c.SourceVersion, "gbak.exe")) {
			return fmt.Errorf("não encontrei o gbak.exe do Firebird %s da rotina", c.SourceVersion)
		}
	}
	if c.Routine.Kind == routineRepair || c.Routine.Kind == routineRepairAdvanced {
		if !fileExists(firebirdExe(c.FbSource, c.SourceVersion, "gfix.exe")) {
			return fmt.Errorf("não encontrei o gfix.exe do Firebird %s da rotina", c.SourceVersion)
		}
		if c.BackupDir == "" || !dirExists(c.BackupDir) {
			return errors.New("selecione uma pasta valida para salvar o backup")
		}
		if _, err := strconv.Atoi(c.SourcePort); err != nil {
			return errors.New("a porta do Firebird da rotina precisa ser numerica")
		}
	}
	if c.TargetVersion != "" {
		if c.TargetVersion == "" {
			return errors.New("selecione a versão de restauração")
		}
		if c.Dest == "" {
			return errors.New("informe o arquivo de destino")
		}
		if strings.EqualFold(filepath.Clean(c.Source), filepath.Clean(c.Dest)) {
			return errors.New("o destino precisa ser diferente do arquivo de origem")
		}
		if c.FbTarget == "" || !fileExists(firebirdExe(c.FbTarget, c.TargetVersion, "gbak.exe")) {
			return fmt.Errorf("não encontrei o gbak.exe do Firebird %s da restauração", c.TargetVersion)
		}
		if _, err := strconv.Atoi(c.TargetPort); err != nil {
			return errors.New("a porta do Firebird da restauração precisa ser numérica")
		}
		if c.TargetVersion == "5.0" && !fileExists(firebirdExe(c.FbTarget, c.TargetVersion, "gfix.exe")) {
			return errors.New("não encontrei o gfix.exe do Firebird 5.0 para ajustar o dialeto após a restauração")
		}
	}
	return nil
}

type appPreferences struct {
	RoutineName   string `json:"routine_name"`
	SourceVersion string `json:"source_version"`
	TargetVersion string `json:"target_version"`
	Source        string `json:"source"`
	Dest          string `json:"dest"`
	BackupDir     string `json:"backup_dir"`
	FbSource      string `json:"fb_source"`
	FbTarget      string `json:"fb_target"`
	DetailedLog   bool   `json:"detailed_log"`
}

func preferencesPath() string {
	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base, _ = os.UserHomeDir()
	}
	if strings.TrimSpace(base) == "" {
		return ""
	}
	return filepath.Join(base, "FirebirdToolbox", "preferences.json")
}

func savePreferences(c workflowConfig) {
	path := preferencesPath()
	if path == "" {
		return
	}
	prefs := appPreferences{
		RoutineName:   c.Routine.Name,
		SourceVersion: c.SourceVersion,
		TargetVersion: c.TargetVersion,
		Source:        c.Source,
		Dest:          c.Dest,
		BackupDir:     c.BackupDir,
		FbSource:      c.FbSource,
		FbTarget:      c.FbTarget,
		DetailedLog:   detailedLog,
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0600)
}

func loadPreferences() appPreferences {
	path := preferencesPath()
	if path == "" {
		return appPreferences{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return appPreferences{}
	}
	var prefs appPreferences
	if json.Unmarshal(data, &prefs) != nil {
		return appPreferences{}
	}
	return prefs
}

func preferenceComboIndex(items []string, value string) int {
	for i, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(value)) {
			return i
		}
	}
	return -1
}

func preferenceRoutineIndex(items []string, value string) int {
	if idx := preferenceComboIndex(items, value); idx >= 0 {
		return idx
	}
	// Compatibilidade com preferences.json gravado até a v1.2.25.
	switch strings.ToLower(strings.TrimSpace(value)) {
	case strings.ToLower("Corrigir banco"):
		return 0
	case strings.ToLower("Corrigir banco (avançado)"), strings.ToLower("Corrigir banco (avancado)"):
		return 1
	default:
		return -1
	}
}

func applySavedPreferences() {
	prefs := loadPreferences()
	if prefs.RoutineName == "" && prefs.Source == "" && prefs.BackupDir == "" {
		return
	}
	if idx := preferenceRoutineIndex(routineNames(routines), prefs.RoutineName); idx >= 0 {
		setCombo(idRoutine, idx)
	}
	if idx := preferenceComboIndex(versions, prefs.SourceVersion); idx >= 0 {
		setCombo(idSourceVersion, idx)
	}
	if prefs.TargetVersion == "" {
		setCombo(idRestoreVersion, 0)
	} else if idx := preferenceComboIndex(restoreVersions, prefs.TargetVersion); idx >= 0 {
		setCombo(idRestoreVersion, idx)
	}

	// Aplica primeiro as regras da rotina e depois restaura os caminhos escolhidos
	// pelo operador. Assim a detecção automática continua sendo o fallback.
	routineChanged()
	if prefs.Source != "" && fileExists(prefs.Source) {
		setText(idSource, prefs.Source)
	}
	if prefs.BackupDir != "" && dirExists(prefs.BackupDir) && selectedRoutine().Kind != routineRestore {
		setText(idBackupDir, prefs.BackupDir)
	}
	if prefs.Dest != "" {
		parent := filepath.Dir(prefs.Dest)
		if parent != "." && dirExists(parent) {
			setText(idDest, prefs.Dest)
		}
	}
	if prefs.FbSource != "" && dirExists(prefs.FbSource) && selectedRoutine().Kind != routineRestore {
		setText(idFbSource, prefs.FbSource)
		setText(idSourcePort, detectFirebirdPort(prefs.FbSource, selectedSourceVersion()))
	}
	if prefs.FbTarget != "" && dirExists(prefs.FbTarget) && selectedRestoreVersion() != "" {
		setText(idFbTarget, prefs.FbTarget)
		setText(idTargetPort, detectFirebirdPort(prefs.FbTarget, selectedRestoreVersion()))
	}
	detailedLog = prefs.DetailedLog
	if detailedLog {
		setText(idDetailedLog, "Resumir log")
	} else {
		setText(idDetailedLog, "Ver detalhes")
	}
	updateUIState(false)
}

type preflightReport struct {
	Checks   []string
	Warnings []string
}

func (p preflightReport) text() string {
	var lines []string
	for _, item := range p.Checks {
		lines = append(lines, "✓ "+item)
	}
	for _, item := range p.Warnings {
		lines = append(lines, "⚠ "+item)
	}
	return strings.Join(lines, "\n")
}

func humanBytes(value uint64) string {
	const (
		kb = uint64(1024)
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case value >= gb:
		return fmt.Sprintf("%.1f GB", float64(value)/float64(gb))
	case value >= mb:
		return fmt.Sprintf("%.0f MB", float64(value)/float64(mb))
	case value >= kb:
		return fmt.Sprintf("%.0f KB", float64(value)/float64(kb))
	default:
		return fmt.Sprintf("%d bytes", value)
	}
}

func diskFreeBytes(dir string) (uint64, error) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "" || !dirExists(dir) {
		return 0, errors.New("pasta inexistente")
	}
	var available, total, totalFree uint64
	r, _, callErr := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(u16(dir))),
		uintptr(unsafe.Pointer(&available)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceExW falhou: %v", callErr)
	}
	return available, nil
}

func ensureWritableDir(dir string) error {
	if !dirExists(dir) {
		return errors.New("a pasta não existe")
	}
	f, err := os.CreateTemp(dir, ".firebird-toolbox-write-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}

func checkToolVersion(label, root, selected string, report *preflightReport) error {
	detected := probeFirebirdVersion(root)
	if detected == "" {
		report.Warnings = append(report.Warnings, label+": não foi possível confirmar a versão pelo gbak -z; o caminho será validado novamente durante a execução")
		return nil
	}
	if detected != selected {
		return fmt.Errorf("%s aponta para Firebird %s, mas a rotina está configurada para Firebird %s", label, detected, selected)
	}
	report.Checks = append(report.Checks, label+": Firebird "+detected+" confirmado pelo gbak")
	return nil
}

func checkEndpointPreflight(label, version, root, port string, report *preflightReport) error {
	resolved := strings.TrimSpace(port)
	if !portReachable(resolved) {
		configured := detectFirebirdPort(root, version)
		if configured != "" && portReachable(configured) {
			resolved = configured
			report.Warnings = append(report.Warnings, label+": porta informada não respondeu; a instalação respondeu na porta "+configured)
		} else {
			return fmt.Errorf("%s não está respondendo em 127.0.0.1:%s; inicie o serviço Firebird ou confira a porta", label, port)
		}
	}
	report.Checks = append(report.Checks, label+": serviço acessível em 127.0.0.1:"+resolved)
	return nil
}

func runPreflight(c workflowConfig) (preflightReport, error) {
	var report preflightReport
	info, err := os.Stat(c.Source)
	if err != nil || info.IsDir() {
		return report, errors.New("não foi possível ler o arquivo de origem")
	}
	report.Checks = append(report.Checks, "origem acessível ("+humanBytes(uint64(info.Size()))+")")

	if c.Routine.Kind != routineRestore {
		if err := checkToolVersion("Firebird da rotina", c.FbSource, c.SourceVersion, &report); err != nil {
			return report, err
		}
		if err := checkEndpointPreflight("Firebird da rotina", c.SourceVersion, c.FbSource, c.SourcePort, &report); err != nil {
			return report, err
		}
		if err := checkSourceODS(c, &report); err != nil {
			return report, err
		}
		if err := ensureWritableDir(c.BackupDir); err != nil {
			return report, fmt.Errorf("a pasta de backup não permite gravação: %w", err)
		}
		report.Checks = append(report.Checks, "pasta de backup permite gravação")

		required := uint64(info.Size()) + 256*1024*1024
		if c.Routine.Kind == routineRepairAdvanced {
			required = uint64(info.Size())*3 + 512*1024*1024
		}
		if free, err := diskFreeBytes(c.BackupDir); err == nil {
			if free < required {
				return report, fmt.Errorf("espaço livre insuficiente na pasta de backup: disponível %s, recomendado no mínimo %s", humanBytes(free), humanBytes(required))
			}
			report.Checks = append(report.Checks, "espaço livre para trabalho/backup: "+humanBytes(free))
		} else {
			report.Warnings = append(report.Warnings, "não foi possível medir o espaço livre da pasta de backup")
		}
	}

	if c.TargetVersion != "" {
		parent := filepath.Dir(c.Dest)
		if parent == "." || !dirExists(parent) {
			return report, errors.New("a pasta do arquivo de destino não existe")
		}
		if err := ensureWritableDir(parent); err != nil {
			return report, fmt.Errorf("a pasta do destino não permite gravação: %w", err)
		}
		report.Checks = append(report.Checks, "pasta do destino permite gravação")
		if fileExists(c.Dest) {
			report.Warnings = append(report.Warnings, "o arquivo de destino já existe e exigirá confirmação antes da restauração")
		}
		if err := checkToolVersion("Firebird da restauração", c.FbTarget, c.TargetVersion, &report); err != nil {
			return report, err
		}
		if err := checkEndpointPreflight("Firebird da restauração", c.TargetVersion, c.FbTarget, c.TargetPort, &report); err != nil {
			return report, err
		}
		required := uint64(info.Size()) + 256*1024*1024
		if free, err := diskFreeBytes(parent); err == nil {
			if free < required {
				return report, fmt.Errorf("espaço livre insuficiente no destino: disponível %s, recomendado no mínimo %s", humanBytes(free), humanBytes(required))
			}
			report.Checks = append(report.Checks, "espaço livre no destino: "+humanBytes(free))
		}
	}
	return report, nil
}

func setStatus(text string) {
	setText(idStatus, text)
}

func quoteForLog(s string) string {
	if strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}

func toggleDetailedLog() {
	if isRunning() {
		return
	}
	detailedLog = !detailedLog
	if detailedLog {
		setText(idDetailedLog, "Resumir log")
		setStatus("Log detalhado ativado. O gbak exibira todas as etapas tecnicas.")
	} else {
		setText(idDetailedLog, "Ver detalhes")
		setStatus("Log resumido ativado. O gbak exibira apenas mensagens essenciais.")
	}
}

func effectiveCommandArgs(exe string, args []string) []string {
	effective := append([]string(nil), args...)
	if detailedLog || !strings.EqualFold(filepath.Base(exe), "gbak.exe") {
		return effective
	}
	filtered := effective[:0]
	for _, arg := range effective {
		if strings.EqualFold(arg, "-v") {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func shouldSurfaceSummaryOutput(output string) bool {
	lower := strings.ToLower(output)
	for _, token := range []string{"error", "erro", "warning", "aviso", "falha", "failed", "corrupt", "invalid", "damaged"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func runCommand(ctx context.Context, label, exe string, args ...string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	appendLog(label)
	effectiveArgs := effectiveCommandArgs(exe, args)
	displayArgs := append([]string(nil), effectiveArgs...)
	for i := 0; i < len(displayArgs)-1; i++ {
		if strings.EqualFold(displayArgs[i], "-password") || strings.EqualFold(displayArgs[i], "-pass") {
			displayArgs[i+1] = "********"
		}
	}

	if !detailedLog {
		output, err := runCommandCapture(ctx, exe, effectiveArgs...)
		if output != "" && (err != nil || shouldSurfaceSummaryOutput(output)) {
			appendLog(output)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	appendLog("  " + quoteForLog(exe) + " " + strings.Join(displayArgs, " "))
	cmd := exec.CommandContext(ctx, exe, effectiveArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return err
	}

	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			line := append([]byte(nil), scanner.Bytes()...)
			appendLog(decodeWindowsBytes(line))
		}
	}()

	err := cmd.Wait()
	_ = pw.Close()
	<-scanDone
	_ = pr.Close()

	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func databaseConnection(port, path string) string {
	return "127.0.0.1/" + port + ":" + path
}

func serviceManager(port string) string {
	if strings.TrimSpace(port) == "" || port == "3050" {
		return "localhost:service_mgr"
	}
	return "localhost/" + port + ":service_mgr"
}

func backupFilePath(c workflowConfig) string {
	base := strings.TrimSuffix(filepath.Base(c.Source), filepath.Ext(c.Source))
	if base == "" {
		base = "DBCOM"
	}
	stamp := time.Now().Format("20060102_150405")
	return filepath.Join(c.BackupDir, base+"_"+stamp+".BAK")
}

type databaseHeader struct {
	ODS               string
	Dialect           string
	PageSize          string
	OldestTransaction string
	OldestActive      string
	OldestSnapshot    string
	NextTransaction   string
	SweepInterval     string
}

func parseDatabaseHeader(output string) databaseHeader {
	var h databaseHeader
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "ods version"):
			h.ODS = strings.TrimSpace(line[len("ODS version"):])
		case strings.HasPrefix(lower, "database dialect"):
			h.Dialect = strings.TrimSpace(line[len("Database dialect"):])
		case strings.HasPrefix(lower, "page size"):
			h.PageSize = strings.TrimSpace(line[len("Page size"):])
		case strings.HasPrefix(lower, "oldest transaction"):
			h.OldestTransaction = strings.TrimSpace(line[len("Oldest transaction"):])
		case strings.HasPrefix(lower, "oldest active"):
			h.OldestActive = strings.TrimSpace(line[len("Oldest active"):])
		case strings.HasPrefix(lower, "oldest snapshot"):
			h.OldestSnapshot = strings.TrimSpace(line[len("Oldest snapshot"):])
		case strings.HasPrefix(lower, "next transaction"):
			h.NextTransaction = strings.TrimSpace(line[len("Next transaction"):])
		case strings.HasPrefix(lower, "sweep interval"):
			h.SweepInterval = strings.TrimSpace(line[len("Sweep interval"):])
		}
	}
	return h
}

// odsFirebirdVersion traduz o ODS gravado no header do banco para a versao do
// Firebird que o criou. O numero do ODS nao muda com service pack, entao e o
// unico dado do arquivo que diz com seguranca qual versao consegue abri-lo.
var odsFirebirdVersion = map[string]string{
	"10.0": "1.0",
	"10.1": "1.5",
	"11.0": "2.0",
	"11.1": "2.1",
	"11.2": "2.5",
	"12.0": "3.0",
	"13.0": "4.0",
	"13.1": "5.0",
}

// parseUnsupportedODS le a recusa que gfix e gstat emitem quando o arquivo foi
// criado por uma versao mais nova do que a do utilitario, por exemplo
// "unsupported on-disk structure for file X; found 13.0, support 11.2".
func parseUnsupportedODS(output string) (found, support string) {
	lower := strings.ToLower(output)
	if !strings.Contains(lower, "unsupported on-disk structure") {
		return "", ""
	}
	idx := strings.Index(lower, "found ")
	if idx < 0 {
		return "", ""
	}
	rest := output[idx+len("found "):]
	found = strings.TrimSpace(strings.SplitN(rest, ",", 2)[0])
	if sup := strings.Index(strings.ToLower(rest), "support "); sup >= 0 {
		tail := rest[sup+len("support "):]
		support = strings.TrimSpace(strings.Trim(strings.SplitN(tail, "\n", 2)[0], " \r.;"))
	}
	return found, support
}

// unsupportedODSMessage transforma a recusa tecnica do Firebird em uma
// instrucao acionavel, dizendo qual versao o operador deve selecionar.
func unsupportedODSMessage(selected, found, support string) string {
	message := "o banco de origem e ODS " + found
	if version := odsFirebirdVersion[found]; version != "" {
		message += " (Firebird " + version + ")"
	}
	message += ", mas o Firebird " + selected + " selecionado em \"Versao da rotina\""
	if support != "" {
		message += " so abre ate ODS " + support
	} else {
		message += " nao consegue abri-lo"
	}
	if version := odsFirebirdVersion[found]; version != "" {
		message += "; selecione Firebird " + version + " e aponte a instalacao correspondente"
	} else {
		message += "; selecione a versao do Firebird que criou o banco"
	}
	return message
}

// toolFailureDetail extrai da saida do utilitario Firebird a primeira linha
// util para o operador. Sem ela o usuario recebe apenas "exit status 1" e a
// causa real fica escondida no log tecnico.
func toolFailureDetail(output string) string {
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "COMANDO:") {
			continue
		}
		return line
	}
	return ""
}

// toolFailure junta a mensagem de contexto, a linha que o utilitario escreveu e
// o codigo de saida, preservando o erro original para quem usar errors.Is.
func toolFailure(message string, err error, output string) error {
	detail := toolFailureDetail(output)
	switch {
	case detail != "" && err != nil:
		return fmt.Errorf("%s: %s (%w)", message, detail, err)
	case detail != "":
		return errors.New(message + ": " + detail)
	case err != nil:
		return fmt.Errorf("%s: %w", message, err)
	default:
		return errors.New(message)
	}
}

// checkSourceODS pede ao proprio Firebird selecionado que leia o header do
// banco. E a unica checagem confiavel de compatibilidade: se essa instalacao
// nao abre o arquivo, a rotina inteira falharia mais adiante, depois de ja ter
// criado pastas e copias.
func checkSourceODS(c workflowConfig, report *preflightReport) error {
	gstat := firebirdExe(c.FbSource, c.SourceVersion, "gstat.exe")
	if !fileExists(gstat) {
		report.Warnings = append(report.Warnings, "gstat.exe nao foi encontrado; o ODS do banco de origem nao pode ser conferido antes da execucao")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := runCommandCapture(ctx, gstat, "-h", c.Source)
	if found, support := parseUnsupportedODS(output); found != "" {
		return errors.New(unsupportedODSMessage(c.SourceVersion, found, support))
	}
	if err != nil {
		report.Warnings = append(report.Warnings, "nao foi possivel ler o header do banco de origem com gstat; a compatibilidade sera conferida durante a execucao")
		return nil
	}
	h := parseDatabaseHeader(output)
	if h.ODS == "" {
		report.Warnings = append(report.Warnings, "o gstat nao informou o ODS do banco de origem")
		return nil
	}
	check := "banco de origem legivel pelo Firebird " + c.SourceVersion + " | ODS " + h.ODS
	if version := odsFirebirdVersion[h.ODS]; version != "" {
		check += " (Firebird " + version + ")"
	}
	report.Checks = append(report.Checks, check)
	return nil
}

func validateRestoredHeader(ctx context.Context, c workflowConfig) error {
	gstat := firebirdExe(c.FbTarget, c.TargetVersion, "gstat.exe")
	if !fileExists(gstat) {
		appendLog("Banco restaurado; gstat.exe n\u00e3o foi encontrado para conferir o header.")
		return nil
	}
	appendLog("Conferindo banco restaurado...")
	output, err := runCommandCapture(ctx, gstat, "-h", c.Dest)
	if detailedLog {
		appendLog("  " + quoteForLog(gstat) + " -h " + quoteForLog(c.Dest))
		if output != "" {
			appendLog(output)
		}
	}
	if err != nil {
		if !detailedLog && output != "" {
			appendLog(output)
		}
		return fmt.Errorf("falha ao conferir o banco restaurado com gstat: %w", err)
	}
	h := parseDatabaseHeader(output)
	setLastHeader(h)
	parts := []string{"\u2713 Banco restaurado validado"}
	if h.ODS != "" {
		parts = append(parts, "ODS "+h.ODS)
	}
	if h.Dialect != "" {
		parts = append(parts, "Dialeto "+h.Dialect)
	}
	if h.PageSize != "" {
		parts = append(parts, "P\u00e1gina "+h.PageSize+" bytes")
	}
	appendLog(strings.Join(parts, " | ") + ".")
	return nil
}

type healthRowCount struct {
	Relation  string
	Reference int64
	Final     int64
	Checked   bool
	Match     bool
	Error     string
}

type postRecoveryHealth struct {
	Checked             bool
	DatabasePath        string
	ReferencePath       string
	Version             string
	Header              databaseHeader
	ConnectivityOK      bool
	SweepOK             bool
	ValidationOK        bool
	Validation          recoveryStats
	ValidationLegacy    bool
	LimboCount          int
	UserTableCount      int64
	UserIndexCount      int64
	PrimaryUniqueCount  int64
	ReferenceTableCount int64
	ReferenceIndexCount int64
	ReferenceConstraint int64
	InactiveIndexes     []string
	InactiveConstraints []string
	RowCounts           []healthRowCount
	CriticalIssues      []string
	Warnings            []string
	Readiness           string
}

func markerInt64(output, prefix string) (int64, bool) {
	items := markerLines(output, prefix)
	if len(items) == 0 {
		return 0, false
	}
	v, err := strconv.ParseInt(strings.TrimSpace(items[len(items)-1]), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func healthMetadataScript() string {
	// Evita COALESCE aqui para manter compatibilidade com Firebird 1.0;
	// COALESCE só foi incorporado à engine a partir do Firebird 1.5.
	return "SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\n" +
		"SELECT 'FTB_HEALTH_PING|OK' FROM RDB$DATABASE;\n" +
		"SELECT 'FTB_HEALTH_TABLES|' || CAST(COUNT(*) AS VARCHAR(30)) FROM RDB$RELATIONS WHERE (RDB$SYSTEM_FLAG IS NULL OR RDB$SYSTEM_FLAG = 0) AND RDB$VIEW_BLR IS NULL;\n" +
		"SELECT 'FTB_HEALTH_INDEXES|' || CAST(COUNT(*) AS VARCHAR(30)) FROM RDB$INDICES WHERE (RDB$SYSTEM_FLAG IS NULL OR RDB$SYSTEM_FLAG = 0);\n" +
		"SELECT 'FTB_HEALTH_CONSTRAINTS|' || CAST(COUNT(*) AS VARCHAR(30)) FROM RDB$RELATION_CONSTRAINTS WHERE RDB$CONSTRAINT_TYPE IN ('PRIMARY KEY','UNIQUE');\n" +
		"SELECT 'FTB_HEALTH_INACTIVE_INDEX|' || RDB$INDEX_NAME FROM RDB$INDICES WHERE (RDB$SYSTEM_FLAG IS NULL OR RDB$SYSTEM_FLAG = 0) AND RDB$INDEX_INACTIVE = 1 ORDER BY RDB$INDEX_NAME;\n" +
		"SELECT 'FTB_HEALTH_INACTIVE_CONSTRAINT|' || rc.RDB$CONSTRAINT_NAME || '|' || rc.RDB$RELATION_NAME || '|' || rc.RDB$CONSTRAINT_TYPE || '|' || rc.RDB$INDEX_NAME FROM RDB$RELATION_CONSTRAINTS rc JOIN RDB$INDICES i ON i.RDB$INDEX_NAME = rc.RDB$INDEX_NAME WHERE rc.RDB$CONSTRAINT_TYPE IN ('PRIMARY KEY','UNIQUE') AND i.RDB$INDEX_INACTIVE = 1 ORDER BY rc.RDB$RELATION_NAME, rc.RDB$CONSTRAINT_NAME;\n" +
		"QUIT;\n"
}

func loadHealthMetadata(ctx context.Context, root, version, port, user, password, dbPath string) (postRecoveryHealth, error) {
	var h postRecoveryHealth
	out, err := runISQLScript(ctx, root, version, port, user, password, dbPath, "Conferindo metadados e índices do banco pós-recuperação...", healthMetadataScript())
	if err != nil {
		return h, err
	}
	h.ConnectivityOK = len(markerLines(out, "FTB_HEALTH_PING|")) > 0
	if n, ok := markerInt64(out, "FTB_HEALTH_TABLES|"); ok {
		h.UserTableCount = n
	}
	if n, ok := markerInt64(out, "FTB_HEALTH_INDEXES|"); ok {
		h.UserIndexCount = n
	}
	if n, ok := markerInt64(out, "FTB_HEALTH_CONSTRAINTS|"); ok {
		h.PrimaryUniqueCount = n
	}
	h.InactiveIndexes = markerLines(out, "FTB_HEALTH_INACTIVE_INDEX|")
	for i := range h.InactiveIndexes {
		h.InactiveIndexes[i] = strings.TrimSpace(h.InactiveIndexes[i])
	}
	for _, item := range markerLines(out, "FTB_HEALTH_INACTIVE_CONSTRAINT|") {
		parts := strings.Split(item, "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		h.InactiveConstraints = append(h.InactiveConstraints, strings.Join(parts, "|"))
	}
	return h, nil
}

func healthRelevantRelations(r *recoveryReport) []string {
	seen := map[string]bool{}
	var result []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToUpper(value)
		if seen[key] {
			return
		}
		seen[key] = true
		result = append(result, value)
	}
	add(r.RowRescueRelation)
	add(r.SuspectedRelation)
	for _, conflict := range r.DuplicateConflicts {
		add(conflict.Relation)
	}
	return result
}

func healthRowCountIdentifier(relation string) string {
	value := strings.TrimSpace(relation)
	if value != "" && value == strings.ToUpper(value) {
		regular := true
		for i, r := range value {
			if i == 0 {
				regular = (r >= 'A' && r <= 'Z') || r == '_'
			} else {
				regular = (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '$'
			}
			if !regular {
				break
			}
		}
		if regular {
			// Compatibilidade essencial com SQL Dialect 1: nomes tradicionais em
			// maiúsculas não devem ser envolvidos em aspas duplas. No caso real,
			// FROM "DOCIT" foi interpretado de forma incompatível e gerou -104.
			return value
		}
	}
	// Identificadores não tradicionais só são esperados em bancos modernos;
	// preserve a forma delimitada para não alterar nomes case-sensitive.
	return sqlIdentifier(value)
}

func healthRowCountScript(relation string) string {
	// LIST ON fornece um rótulo estável sem depender de concatenação/cast apenas
	// para marcar a saída. PLAN NATURAL mantém a leitura física completa usada
	// pelo gate de conferência pós-recuperação.
	return "SET HEADING OFF;\nSET LIST ON;\nSET COUNT OFF;\nSELECT COUNT(*) AS FTB_HEALTH_ROWCOUNT FROM " + healthRowCountIdentifier(relation) + " T PLAN (T NATURAL);\nQUIT;\n"
}

func parseHealthRowCount(output string) (int64, bool) {
	const marker = "FTB_HEALTH_ROWCOUNT"
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) < len(marker) || !strings.EqualFold(line[:len(marker)], marker) {
			continue
		}
		rest := strings.TrimSpace(line[len(marker):])
		rest = strings.TrimLeft(rest, "=:| \t")
		for _, field := range strings.Fields(rest) {
			field = strings.Trim(field, "|,; \t")
			if n, err := strconv.ParseInt(field, 10, 64); err == nil {
				return n, true
			}
		}
	}
	return 0, false
}

func countRelationRows(ctx context.Context, root, version, port, user, password, dbPath, relation string) (int64, error) {
	script := healthRowCountScript(relation)
	out, err := runISQLScript(ctx, root, version, port, user, password, dbPath, "Conferindo quantidade de registros em "+relation+"...", script)
	if err != nil {
		return 0, fmt.Errorf("consulta de contagem de %s falhou: %w", relation, err)
	}
	n, ok := parseHealthRowCount(out)
	if !ok {
		return 0, fmt.Errorf("não foi possível interpretar a contagem de %s", relation)
	}
	return n, nil
}

func headerTransactionGap(h databaseHeader) (int64, bool) {
	next, err1 := strconv.ParseInt(strings.TrimSpace(h.NextTransaction), 10, 64)
	oit, err2 := strconv.ParseInt(strings.TrimSpace(h.OldestTransaction), 10, 64)
	if err1 != nil || err2 != nil || next < oit {
		return 0, false
	}
	return next - oit, true
}

func appendHealthIssue(h *postRecoveryHealth, issue string) {
	issue = strings.TrimSpace(issue)
	if issue != "" {
		h.CriticalIssues = append(h.CriticalIssues, issue)
	}
}

func appendHealthWarning(h *postRecoveryHealth, warning string) {
	warning = strings.TrimSpace(warning)
	if warning != "" {
		h.Warnings = append(h.Warnings, warning)
	}
}

func writeHealthReport(path string, h postRecoveryHealth, partial bool, hasFinalDestination bool) error {
	if path == "" {
		return errors.New("caminho do relatório de saúde não definido")
	}
	var b strings.Builder
	b.WriteString("FIREBIRD TOOLBOX - RELATÓRIO DE SAÚDE E PÓS-RECUPERAÇÃO\r\n")
	b.WriteString("Versão do utilitário: " + appVersion + "\r\n")
	b.WriteString("Banco avaliado: " + h.DatabasePath + "\r\n")
	b.WriteString("Firebird: " + h.Version + "\r\n")
	if h.ReferencePath != "" {
		b.WriteString("Restore de referência: " + h.ReferencePath + "\r\n")
	}

	b.WriteString("\r\nINTEGRIDADE E ACESSO\r\n")
	b.WriteString(fmt.Sprintf("Banco acessível via SQL: %t\r\n", h.ConnectivityOK))
	b.WriteString(fmt.Sprintf("Sweep final concluído: %t\r\n", h.SweepOK))
	b.WriteString(fmt.Sprintf("Validação estrutural aprovada: %t\r\n", h.ValidationOK))
	if h.ValidationLegacy {
		b.WriteString("Validação legada: o gfix não reconheceu -no_update; a validação foi executada somente no banco restaurado.\r\n")
	}
	b.WriteString("Diagnóstico estrutural final: " + h.Validation.Summary() + "\r\n")
	b.WriteString(fmt.Sprintf("Transações em limbo: %d\r\n", h.LimboCount))

	b.WriteString("\r\nHEADER / TRANSAÇÕES\r\n")
	if h.Header.ODS != "" {
		b.WriteString("ODS: " + h.Header.ODS + "\r\n")
	}
	if h.Header.Dialect != "" {
		b.WriteString("Dialeto: " + h.Header.Dialect + "\r\n")
	}
	if h.Header.PageSize != "" {
		b.WriteString("Page size: " + h.Header.PageSize + " bytes\r\n")
	}
	if h.Header.OldestTransaction != "" {
		b.WriteString("Oldest transaction: " + h.Header.OldestTransaction + "\r\n")
	}
	if h.Header.OldestActive != "" {
		b.WriteString("Oldest active: " + h.Header.OldestActive + "\r\n")
	}
	if h.Header.OldestSnapshot != "" {
		b.WriteString("Oldest snapshot: " + h.Header.OldestSnapshot + "\r\n")
	}
	if h.Header.NextTransaction != "" {
		b.WriteString("Next transaction: " + h.Header.NextTransaction + "\r\n")
	}
	if h.Header.SweepInterval != "" {
		b.WriteString("Sweep interval: " + h.Header.SweepInterval + "\r\n")
	}
	if gap, ok := headerTransactionGap(h.Header); ok {
		b.WriteString(fmt.Sprintf("Gap Next/OIT: %d\r\n", gap))
	}

	b.WriteString("\r\nMETADADOS / ÍNDICES\r\n")
	b.WriteString(fmt.Sprintf("Tabelas de usuário: %d\r\n", h.UserTableCount))
	b.WriteString(fmt.Sprintf("Índices de usuário: %d\r\n", h.UserIndexCount))
	b.WriteString(fmt.Sprintf("Constraints PRIMARY KEY/UNIQUE: %d\r\n", h.PrimaryUniqueCount))
	if h.ReferencePath != "" {
		b.WriteString(fmt.Sprintf("Referência - tabelas: %d | índices: %d | PK/UNIQUE: %d\r\n", h.ReferenceTableCount, h.ReferenceIndexCount, h.ReferenceConstraint))
	}
	if len(h.InactiveIndexes) == 0 {
		b.WriteString("Índices inativos: 0\r\n")
	} else {
		b.WriteString("Índices inativos: " + strings.Join(h.InactiveIndexes, ", ") + "\r\n")
	}
	if len(h.InactiveConstraints) == 0 {
		b.WriteString("PRIMARY KEY/UNIQUE inativas: 0\r\n")
	} else {
		b.WriteString("PRIMARY KEY/UNIQUE inativas: " + strings.Join(h.InactiveConstraints, "; ") + "\r\n")
	}

	if len(h.RowCounts) > 0 {
		b.WriteString("\r\nCONTAGENS ESSENCIAIS\r\n")
		for _, rc := range h.RowCounts {
			if rc.Error != "" {
				b.WriteString(rc.Relation + ": ERRO - " + rc.Error + "\r\n")
			} else if rc.Checked {
				b.WriteString(fmt.Sprintf("%s: final=%d | referência=%d | confere=%t\r\n", rc.Relation, rc.Final, rc.Reference, rc.Match))
			} else {
				b.WriteString(fmt.Sprintf("%s: final=%d\r\n", rc.Relation, rc.Final))
			}
		}
	}

	if len(h.Warnings) > 0 {
		b.WriteString("\r\nAVISOS\r\n")
		for _, item := range h.Warnings {
			b.WriteString("- " + item + "\r\n")
		}
	}
	if len(h.CriticalIssues) > 0 {
		b.WriteString("\r\nGATES CRÍTICOS REPROVADOS\r\n")
		for _, item := range h.CriticalIssues {
			b.WriteString("- " + item + "\r\n")
		}
	}

	b.WriteString("\r\nRESULTADO\r\n")
	b.WriteString("Estado de saúde: " + h.Readiness + "\r\n")
	if partial && len(h.CriticalIssues) == 0 {
		b.WriteString("Observação: a estrutura passou nos gates, porém houve recuperação parcial/decisão de dados; exige homologação funcional antes de produção.\r\n")
	}
	if !hasFinalDestination && len(h.CriticalIssues) == 0 {
		b.WriteString("Observação: a análise foi feita no restore de teste; nenhum banco final foi liberado para produção nesta execução.\r\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0600)
}

func runPostRecoveryHealth(ctx context.Context, c workflowConfig, dbPath, referencePath, healthPath string, report *recoveryReport) (postRecoveryHealth, error) {
	var h postRecoveryHealth
	h.Checked = true
	h.DatabasePath = dbPath
	h.ReferencePath = referencePath
	version, root, port := testRestoreEngine(c)
	if c.TargetVersion != "" && strings.EqualFold(filepath.Clean(dbPath), filepath.Clean(c.Dest)) {
		version, root, port = c.TargetVersion, c.FbTarget, c.TargetPort
	}
	h.Version = version

	setStatus("Executando pós-recuperação e gates de saúde...")
	setProgress(94, "Pós-recuperação: verificando saúde...")
	resolvedPort, err := verifyFirebirdEndpoint(ctx, "do pós-recuperação", version, root, "", port)
	if err != nil {
		appendHealthIssue(&h, "não foi possível confirmar a instância Firebird do banco restaurado: "+err.Error())
		h.Readiness = "NÃO APTO PARA PRODUÇÃO"
		_ = writeHealthReport(healthPath, h, report.PartialRecovery, c.TargetVersion != "")
		return h, errors.New("pós-recuperação não conseguiu acessar a instância Firebird")
	}

	meta, metaErr := loadHealthMetadata(ctx, root, version, resolvedPort, c.User, c.Password, dbPath)
	if metaErr != nil {
		appendHealthIssue(&h, "banco não pôde ser aberto por SQL no pós-recuperação: "+metaErr.Error())
	} else {
		h.ConnectivityOK = meta.ConnectivityOK
		h.UserTableCount = meta.UserTableCount
		h.UserIndexCount = meta.UserIndexCount
		h.PrimaryUniqueCount = meta.PrimaryUniqueCount
		h.InactiveIndexes = meta.InactiveIndexes
		h.InactiveConstraints = meta.InactiveConstraints
		if !h.ConnectivityOK {
			appendHealthIssue(&h, "o banco não respondeu ao SELECT de verificação")
		}
		if len(h.InactiveIndexes) > 0 {
			appendHealthIssue(&h, fmt.Sprintf("%d índice(s) de usuário permanecem inativos: %s", len(h.InactiveIndexes), strings.Join(h.InactiveIndexes, ", ")))
		}
		if len(h.InactiveConstraints) > 0 {
			appendHealthIssue(&h, fmt.Sprintf("%d constraint(s) PRIMARY KEY/UNIQUE permanecem inativas: %s", len(h.InactiveConstraints), strings.Join(h.InactiveConstraints, "; ")))
		}
	}

	if referencePath != "" && fileExists(referencePath) && !strings.EqualFold(filepath.Clean(referencePath), filepath.Clean(dbPath)) {
		refMeta, refErr := loadHealthMetadata(ctx, root, version, resolvedPort, c.User, c.Password, referencePath)
		if refErr != nil {
			appendHealthWarning(&h, "não foi possível comparar metadados com o restore de referência: "+refErr.Error())
		} else {
			h.ReferenceTableCount = refMeta.UserTableCount
			h.ReferenceIndexCount = refMeta.UserIndexCount
			h.ReferenceConstraint = refMeta.PrimaryUniqueCount
			if h.UserTableCount != h.ReferenceTableCount {
				appendHealthIssue(&h, fmt.Sprintf("quantidade de tabelas divergiu do restore de referência: final=%d referência=%d", h.UserTableCount, h.ReferenceTableCount))
			}
			if h.UserIndexCount != h.ReferenceIndexCount {
				appendHealthIssue(&h, fmt.Sprintf("quantidade de índices divergiu do restore de referência: final=%d referência=%d", h.UserIndexCount, h.ReferenceIndexCount))
			}
			if h.PrimaryUniqueCount != h.ReferenceConstraint {
				appendHealthIssue(&h, fmt.Sprintf("quantidade de PRIMARY KEY/UNIQUE divergiu do restore de referência: final=%d referência=%d", h.PrimaryUniqueCount, h.ReferenceConstraint))
			}
		}
	}

	gfix := firebirdExe(root, version, "gfix.exe")
	conn := databaseConnection(resolvedPort, dbPath)
	limboOutput, limboErr := runRecoveryCapture(ctx, "Pós-recuperação: conferindo transações em limbo...", gfix, "-user", c.User, "-pass", c.Password, "-list", conn)
	if limboErr != nil && toolOutputHasOperationalFailure(limboOutput) {
		appendHealthIssue(&h, "não foi possível concluir a verificação de transações em limbo")
	} else {
		h.LimboCount = countLimboTransactions(limboOutput)
		if h.LimboCount > 0 {
			appendHealthIssue(&h, fmt.Sprintf("%d transação(ões) em limbo permanecem no banco restaurado", h.LimboCount))
		}
	}

	setProgress(96, "Pós-recuperação: executando sweep final...")
	sweepOutput, sweepErr := runRecoveryCapture(ctx, "Pós-recuperação: executando sweep final...", gfix, "-user", c.User, "-pass", c.Password, "-sweep", conn)
	if sweepErr != nil || toolOutputHasOperationalFailure(sweepOutput) {
		appendHealthIssue(&h, "sweep final não foi concluído com segurança")
	} else {
		h.SweepOK = true
	}

	setProgress(97, "Pós-recuperação: validação estrutural final...")
	validateArgs := []string{"-user", c.User, "-pass", c.Password, "-v", "-full", "-no_update", conn}
	validationOutput, validationErr := runRecoveryCapture(ctx, "Pós-recuperação: validação estrutural final (somente leitura)...", gfix, validateArgs...)
	if outputReportsUnknownSwitch(validationOutput) {
		h.ValidationLegacy = true
		appendHealthWarning(&h, "o gfix desta versão não reconhece -no_update; a validação final foi repetida somente no banco restaurado usando o modo legado")
		validationOutput, validationErr = runRecoveryCapture(ctx, "Pós-recuperação: validação estrutural final (modo legado, somente no restaurado)...", gfix, "-user", c.User, "-pass", c.Password, "-v", "-full", conn)
	}
	h.Validation = parseRecoveryStats(validationOutput)
	if validationErr != nil && !h.Validation.HasCorruption() {
		appendHealthIssue(&h, "gfix não conseguiu concluir a validação estrutural final")
	} else if toolOutputHasOperationalFailure(validationOutput) && !h.Validation.HasCorruption() {
		appendHealthIssue(&h, "gfix reportou falha operacional durante a validação estrutural final")
	} else if h.Validation.HasCorruption() {
		appendHealthIssue(&h, "a validação final ainda encontrou corrupção: "+h.Validation.Summary())
	} else {
		h.ValidationOK = true
	}

	gstat := firebirdExe(root, version, "gstat.exe")
	if fileExists(gstat) {
		headerOutput, headerErr := runRecoveryCapture(ctx, "Pós-recuperação: conferindo header e indicadores transacionais...", gstat, "-h", conn)
		if headerErr != nil {
			appendHealthIssue(&h, "gstat não conseguiu ler o header do banco restaurado")
		} else {
			h.Header = parseDatabaseHeader(headerOutput)
			if gap, ok := headerTransactionGap(h.Header); ok {
				if sweepInt, parseErr := strconv.ParseInt(strings.TrimSpace(h.Header.SweepInterval), 10, 64); parseErr == nil && sweepInt > 0 && gap > sweepInt*2 {
					appendHealthWarning(&h, fmt.Sprintf("gap transacional Next/OIT (%d) está acima de duas vezes o sweep interval (%d); acompanhe após colocar o banco em uso", gap, sweepInt))
				}
			}
		}
	} else {
		appendHealthWarning(&h, "gstat.exe não foi encontrado; ODS e indicadores transacionais não puderam ser reconferidos")
	}

	for _, relation := range healthRelevantRelations(report) {
		rc := healthRowCount{Relation: relation}
		finalCount, countErr := countRelationRows(ctx, root, version, resolvedPort, c.User, c.Password, dbPath, relation)
		if countErr != nil {
			rc.Error = countErr.Error()
			appendHealthIssue(&h, "não foi possível contar a relação crítica "+relation+": "+countErr.Error())
			h.RowCounts = append(h.RowCounts, rc)
			continue
		}
		rc.Final = finalCount
		if referencePath != "" && fileExists(referencePath) && !strings.EqualFold(filepath.Clean(referencePath), filepath.Clean(dbPath)) {
			refCount, refErr := countRelationRows(ctx, root, version, resolvedPort, c.User, c.Password, referencePath, relation)
			if refErr != nil {
				rc.Error = "referência: " + refErr.Error()
				appendHealthIssue(&h, "não foi possível conferir a relação crítica "+relation+" no restore de referência")
			} else {
				rc.Reference = refCount
				rc.Checked = true
				rc.Match = refCount == finalCount
				if !rc.Match {
					appendHealthIssue(&h, fmt.Sprintf("contagem da relação %s divergiu: final=%d referência=%d", relation, finalCount, refCount))
				}
			}
		}
		h.RowCounts = append(h.RowCounts, rc)
	}

	if h.Header.ODS != "" || h.Header.Dialect != "" || h.Header.PageSize != "" {
		setLastHeader(h.Header)
	}

	// Um banco restaurado com -n perdeu as FOREIGN KEY e as constraints CHECK.
	// Ele pode passar em todos os gates estruturais e ainda assim aceitar dados
	// invalidos, entao nunca recebe APTO PARA PRODUCAO.
	if report.NoValidityRestore {
		appendHealthWarning(&h, "banco restaurado com -n (-no_validity): as FOREIGN KEY e as constraints CHECK não existem e precisam ser recriadas manualmente após sanear as linhas órfãs")
	}

	if len(h.CriticalIssues) > 0 {
		h.Readiness = "NÃO APTO PARA PRODUÇÃO"
	} else if report.NoValidityRestore {
		h.Readiness = "RESTAURADO SEM CONSTRAINTS (-n); RECRIAR FOREIGN KEY ANTES DE USAR"
	} else if report.PartialRecovery {
		h.Readiness = "ESTRUTURALMENTE SAUDÁVEL; REVISÃO FUNCIONAL OBRIGATÓRIA"
	} else if c.TargetVersion != "" {
		h.Readiness = "APTO PARA PRODUÇÃO"
	} else {
		h.Readiness = "RESTORE DE TESTE SAUDÁVEL"
	}
	if err := writeHealthReport(healthPath, h, report.PartialRecovery, c.TargetVersion != ""); err != nil {
		appendHealthWarning(&h, "não foi possível gravar o relatório de saúde: "+err.Error())
	}
	if len(h.CriticalIssues) > 0 {
		return h, fmt.Errorf("pós-recuperação reprovou %d gate(s) crítico(s); banco não foi liberado para produção", len(h.CriticalIssues))
	}
	return h, nil
}

func restoreBackup(ctx context.Context, c workflowConfig, backup string, noValidity bool) error {
	version := c.TargetVersion
	gbak := firebirdExe(c.FbTarget, version, "gbak.exe")

	setStatus("Restaurando para Firebird " + version + "...")
	setProgress(68, "Restaurando backup...")
	appendLog("Restaurando backup para Firebird " + version + ".")
	targetPort := c.TargetPort
	useService := supportsServiceManagerRestore(version)
	if useService {
		resolvedPort, err := verifyFirebirdEndpoint(ctx, "da restauração", version, c.FbTarget, "", targetPort)
		if err != nil {
			return err
		}
		targetPort = resolvedPort
	} else {
		appendLog("Firebird 1.0: usando restauração local por compatibilidade com instalações legadas.")
	}

	restoreAction := "-c"
	if fileExists(c.Dest) {
		restoreAction = "-rep"
	}
	restoreFlags := []string{restoreAction, "-v"}
	// Se o restore de teste so passou com -n, o destino final tem de usar o
	// mesmo -n, senao ele falha exatamente na mesma FOREIGN KEY.
	if noValidity {
		restoreFlags = append(restoreFlags, "-n")
		appendLog("Restaurando o destino final com -n (-no_validity): o banco ficará SEM FOREIGN KEY e SEM constraints CHECK.")
	}
	var args []string
	if useService {
		args = append(restoreFlags, "-se", serviceManager(targetPort), "-user", c.User, "-password", c.Password, backup, c.Dest)
	} else {
		args = append(restoreFlags, "-user", c.User, "-password", c.Password, backup, c.Dest)
	}
	if err := runCommand(ctx, "Executando restauração...", gbak, args...); err != nil {
		return fmt.Errorf("falha na restauração Firebird %s: %w", version, err)
	}
	if !fileExists(c.Dest) {
		return fmt.Errorf("a restauração terminou sem criar o arquivo de destino esperado: %s", c.Dest)
	}
	appendLog("\u2713 Restauração concluída.")
	setProgress(84, "Restauração concluída")

	if version == "5.0" {
		gfix := firebirdExe(c.FbTarget, version, "gfix.exe")
		if err := runCommand(ctx, "Ajustando dialeto do banco restaurado para 1...", gfix, "-user", c.User, "-password", c.Password, "-sql_dialect", "1", c.Dest); err != nil {
			return fmt.Errorf("restauração concluída, mas falhou o ajuste de dialeto no Firebird 5.0: %w", err)
		}
		appendLog("\u2713 Dialeto ajustado para 1.")
		setProgress(91, "Dialeto ajustado")
	}
	setProgress(95, "Validando resultado...")
	if err := validateRestoredHeader(ctx, c); err != nil {
		return err
	}
	appendLog("Resultado final: Firebird " + version + " | " + c.Dest)
	return nil
}

type recoveryStats struct {
	RecordErrors      int
	IndexErrors       int
	TransactionErrors int
	BlobErrors        int
	DatabaseErrors    int
	GenericCorruption bool
}

func (s recoveryStats) HasCorruption() bool {
	return s.RecordErrors > 0 || s.IndexErrors > 0 || s.TransactionErrors > 0 || s.BlobErrors > 0 || s.DatabaseErrors > 0 || s.GenericCorruption
}

func (s recoveryStats) IndexOnly() bool {
	return s.IndexErrors > 0 && s.RecordErrors == 0 && s.TransactionErrors == 0 && s.BlobErrors == 0 && s.DatabaseErrors == 0 && !s.GenericCorruption
}

func (s recoveryStats) Severity() string {
	switch {
	case s.RecordErrors > 0 || s.BlobErrors > 0 || s.DatabaseErrors > 0 || s.GenericCorruption:
		return "ALTA"
	case s.TransactionErrors > 0:
		return "MÉDIA"
	case s.IndexErrors > 0:
		return "BAIXA (índices)"
	default:
		return "SEM CORRUPÇÃO REPORTADA"
	}
}

func (s recoveryStats) Summary() string {
	if !s.HasCorruption() {
		return "nenhuma corrupção estrutural foi reportada"
	}
	parts := []string{"severidade " + s.Severity()}
	if s.RecordErrors > 0 {
		parts = append(parts, fmt.Sprintf("registros=%d", s.RecordErrors))
	}
	if s.IndexErrors > 0 {
		parts = append(parts, fmt.Sprintf("índices=%d", s.IndexErrors))
	}
	if s.TransactionErrors > 0 {
		parts = append(parts, fmt.Sprintf("transações=%d", s.TransactionErrors))
	}
	if s.BlobErrors > 0 {
		parts = append(parts, fmt.Sprintf("BLOBs=%d", s.BlobErrors))
	}
	if s.DatabaseErrors > 0 {
		parts = append(parts, fmt.Sprintf("páginas=%d", s.DatabaseErrors))
	}
	if s.GenericCorruption && len(parts) == 1 {
		parts = append(parts, "erro estrutural não quantificado")
	}
	return strings.Join(parts, " | ")
}

func countValidationError(output, label string) int {
	pattern := `(?im)^\s*Number of ` + regexp.QuoteMeta(label) + `\s*[:=]?\s*(\d+)\s*$`
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(output)
	if len(m) != 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func parseRecoveryStats(output string) recoveryStats {
	stats := recoveryStats{
		RecordErrors:      countValidationError(output, "record level errors"),
		IndexErrors:       countValidationError(output, "index page errors"),
		TransactionErrors: countValidationError(output, "transaction page errors"),
		BlobErrors:        countValidationError(output, "BLOB errors"),
		DatabaseErrors:    countValidationError(output, "database page errors"),
	}
	lower := strings.ToLower(output)
	for _, token := range []string{
		"database file appears corrupt",
		"database corruption",
		"corrupt data structure",
		"wrong page type",
		"bad checksum",
		"checksum error",
		"page is of wrong type",
	} {
		if strings.Contains(lower, token) {
			stats.GenericCorruption = true
			break
		}
	}
	return stats
}

func mergeRecoveryStats(a, b recoveryStats) recoveryStats {
	max := func(x, y int) int {
		if x > y {
			return x
		}
		return y
	}
	return recoveryStats{
		RecordErrors:      max(a.RecordErrors, b.RecordErrors),
		IndexErrors:       max(a.IndexErrors, b.IndexErrors),
		TransactionErrors: max(a.TransactionErrors, b.TransactionErrors),
		BlobErrors:        max(a.BlobErrors, b.BlobErrors),
		DatabaseErrors:    max(a.DatabaseErrors, b.DatabaseErrors),
		GenericCorruption: a.GenericCorruption || b.GenericCorruption,
	}
}

func outputReportsUnknownSwitch(output string) bool {
	lower := strings.ToLower(output)
	for _, token := range []string{"unknown switch", "invalid switch", "unknown option", "invalid option"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func toolOutputHasOperationalFailure(output string) bool {
	lower := strings.ToLower(output)
	for _, token := range []string{
		"failed to attach",
		"cannot attach",
		"unable to complete network request",
		"unavailable database",
		"i/o error",
		"io error",
		"no permission for",
		"user name and password are not defined",
		"your user name and password are not defined",
		"invalid database handle",
		"database is shutdown",
		"connection rejected",
		"unknown switch",
		"invalid switch",
		"unknown option",
		"invalid option",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func tailText(text string, maxLines int) string {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\n")
	if len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-maxLines:], "\n")
}

func setRecoveryTechnicalLog(path string) {
	recoveryTechnicalLogMu.Lock()
	recoveryTechnicalLog = path
	recoveryTechnicalLogMu.Unlock()
}

func recoveryTechnicalLogPath() string {
	recoveryTechnicalLogMu.Lock()
	defer recoveryTechnicalLogMu.Unlock()
	return recoveryTechnicalLog
}

func recoveryCommandArgs(exe string, args []string) []string {
	// No fluxo avançado, o gbak sempre mantém -v internamente para que seja
	// possível identificar a última relação processada e o erro real. O painel
	// continua resumido; a saída completa fica em LOG_TECNICO_RECUPERACAO.txt.
	return append([]string(nil), args...)
}

func maskedCommandForLog(exe string, args []string) string {
	displayArgs := append([]string(nil), args...)
	for i := 0; i < len(displayArgs)-1; i++ {
		if strings.EqualFold(displayArgs[i], "-password") || strings.EqualFold(displayArgs[i], "-pass") {
			displayArgs[i+1] = "********"
		}
	}
	return quoteForLog(exe) + " " + strings.Join(displayArgs, " ")
}

func runRecoveryCapture(ctx context.Context, label, exe string, args ...string) (string, error) {
	appendLog(label)
	effectiveArgs := recoveryCommandArgs(exe, args)
	if detailedLog {
		appendLog("  " + maskedCommandForLog(exe, effectiveArgs))
	}

	var techFile *os.File
	var techWriter *bufio.Writer
	if techPath := recoveryTechnicalLogPath(); techPath != "" {
		if f, err := os.OpenFile(techPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
			techFile = f
			techWriter = bufio.NewWriterSize(f, 64*1024)
		}
	}
	writeTech := func(text string) {
		if techWriter == nil {
			return
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		stamp := time.Now().Format("15:04:05")
		for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
			_, _ = fmt.Fprintf(techWriter, "[%s] %s\r\n", stamp, line)
		}
	}
	closeTech := func() {
		if techWriter != nil {
			_ = techWriter.Flush()
		}
		if techFile != nil {
			_ = techFile.Close()
		}
	}
	writeTech(label)
	writeTech("COMANDO: " + maskedCommandForLog(exe, effectiveArgs))

	cmd := exec.CommandContext(ctx, exe, effectiveArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		writeTech("ERRO AO INICIAR: " + err.Error())
		closeTech()
		return "", err
	}

	maxCapturedLines := 2000
	if strings.EqualFold(filepath.Base(exe), "gbak.exe") {
		maxCapturedLines = 1200
	}
	captured := make([]string, 0, minInt(maxCapturedLines, 256))
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			line := decodeWindowsBytes(append([]byte(nil), scanner.Bytes()...))
			writeTech(line)
			if detailedLog {
				appendLog(line)
			}
			captured = append(captured, line)
			if len(captured) > maxCapturedLines {
				copy(captured, captured[len(captured)-maxCapturedLines:])
				captured = captured[:maxCapturedLines]
			}
		}
	}()
	err := cmd.Wait()
	_ = pw.Close()
	<-scanDone
	_ = pr.Close()
	output := strings.Join(captured, "\n")
	if err != nil {
		writeTech("PROCESSO FINALIZADO COM ERRO: " + err.Error())
	} else {
		writeTech("PROCESSO FINALIZADO COM SUCESSO")
	}
	closeTech()
	if ctx.Err() != nil {
		return output, ctx.Err()
	}
	return output, err
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type firebirdLogSnapshot struct {
	Path   string
	Offset int64
}

func firebirdLogPath(root string) string {
	root = strings.TrimRight(strings.TrimSpace(root), `\\/`)
	if strings.EqualFold(filepath.Base(root), "bin") {
		root = filepath.Dir(root)
	}
	if root == "" {
		return ""
	}
	return filepath.Join(root, "firebird.log")
}

func snapshotFirebirdLog(root string) firebirdLogSnapshot {
	path := firebirdLogPath(root)
	if path == "" {
		return firebirdLogSnapshot{}
	}
	st, err := os.Stat(path)
	if err != nil {
		return firebirdLogSnapshot{Path: path}
	}
	return firebirdLogSnapshot{Path: path, Offset: st.Size()}
}

func collectFirebirdLogDelta(snap firebirdLogSnapshot) string {
	if snap.Path == "" {
		return ""
	}
	f, err := os.Open(snap.Path)
	if err != nil {
		return ""
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Size() == 0 {
		return ""
	}
	start := snap.Offset
	if start < 0 || start > st.Size() {
		start = 0
	}
	const maxBytes int64 = 128 * 1024
	if st.Size()-start > maxBytes {
		start = st.Size() - maxBytes
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return ""
	}
	data, err := io.ReadAll(io.LimitReader(f, maxBytes))
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(decodeWindowsBytes(data))
	if text == "" {
		return ""
	}
	return tailText(text, 100)
}

func normalizeRelationName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`'\\\"")
	value = strings.TrimRight(value, ".,;:")
	return strings.TrimSpace(value)
}

func plausibleRelationName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	switch strings.ToUpper(name) {
	case "IS", "WAS", "NOT", "FOR", "DATA", "NAME", "WITH", "FROM", "THE", "A", "AN":
		return false
	}
	return true
}

// suspectedRelationFromFirebirdLog devolve a tabela que o motor estava varrendo
// quando sofreu bugcheck. E informacao de diagnostico, nao alvo de -skip_data:
// esse bugcheck costuma vir da fase de validacao do gfix e apontar um sitio de
// dano diferente do que bloqueia o backup. Caso real DBCOM.RED: o firebird.log
// acusava "bugcheck during scan of table 132 (CAIXA)" enquanto o gbak travava
// na pagina 413591 ao processar EMAIL. Pular CAIXA nao mudou nada; pular EMAIL
// levou o backup adiante. Quem escolhe o -skip_data e suspectedRelationFromGbak.
func suspectedRelationFromFirebirdLog(logText string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)bugcheck during scan of table\s+\d+\s*\(([^)]+)\)`),
		regexp.MustCompile(`(?i)has bad transaction\s+\d+\s+in table\s+("[^"]+"|[A-Za-z0-9_$]+)`),
	}
	for _, re := range patterns {
		matches := re.FindAllStringSubmatch(logText, -1)
		for i := len(matches) - 1; i >= 0; i-- {
			if name := normalizeRelationName(matches[i][1]); plausibleRelationName(name) {
				return name
			}
		}
	}
	return ""
}

func suspectedRelationFromGbak(output string) string {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:writing|restoring|processing|backing up)\s+(?:data\s+)?(?:for\s+)?table\s+("[^"]+"|[A-Za-z0-9_$]+)`),
		regexp.MustCompile(`(?i)\brelation\s+("[^"]+"|[A-Za-z0-9_$]+)`),
		regexp.MustCompile(`(?i)\btable\s+("[^"]+"|[A-Za-z0-9_$]+)`),
	}
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		for _, re := range patterns {
			m := re.FindStringSubmatch(line)
			if len(m) == 2 {
				name := normalizeRelationName(m[1])
				if plausibleRelationName(name) {
					return name
				}
			}
		}
	}
	return ""
}

func safeRelationForSkip(name string) bool {
	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, prefix := range []string{"RDB$", "MON$", "SEC$"} {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	return true
}

func sqlRegexLiteral(name string) string {
	name = normalizeRelationName(name)
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '_':
			b.WriteString("[_]")
		case '%':
			b.WriteString("[%]")
		case '[', ']', '(', ')', '|', '*', '+', '?', '{', '}', '^', '$', '.', '\\':
			b.WriteByte('[')
			b.WriteRune(r)
			b.WriteByte(']')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func relevantGbakFailureLines(output string, maxLines int) string {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(output), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	isRelevant := func(line string) bool {
		lower := strings.ToLower(line)
		for _, token := range []string{
			"error", "erro", "failed", "falha", "corrupt", "damaged", "malformed",
			"wrong page", "checksum", "cannot", "can't", "unable", "unexpected",
			"table ", "relation ", "writing data", "restoring data", "record",
		} {
			if strings.Contains(lower, token) {
				return true
			}
		}
		return false
	}
	selected := make([]string, 0, maxLines)
	startTail := len(lines) - 18
	if startTail < 0 {
		startTail = 0
	}
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isRelevant(line) || i >= startTail {
			selected = append(selected, line)
		}
	}
	if len(selected) > maxLines {
		selected = selected[len(selected)-maxLines:]
	}
	return strings.Join(selected, "\n")
}

// gbakLooksLikeIndexCorruption exige que a pagina quebrada seja de indice, ou
// seja, que o indice esteja no lado ESPERADO da mensagem do Firebird. A versao
// anterior aceitava "wrong page type" junto de qualquer ocorrencia de "index"
// na saida, e o gbak escreve "writing index ..." o tempo todo: na pratica
// qualquer falha de tipo de pagina virava suspeita de indice. Caso real: o erro
// era "page 413591 is of wrong type (expected transaction inventory, found
// data)" e o operador recebia um aviso mandando procurar indice para desativar.
// "found index B-tree" tambem nao serve: ali o indice e o invasor, nao a
// pagina danificada.
func gbakLooksLikeIndexCorruption(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "expected index b-tree") ||
		strings.Contains(lower, "expected index") ||
		strings.Contains(lower, "index page errors")
}

// gbakLooksLikeOrphanForeignKey identifica o restore que falhou ao recriar uma
// FOREIGN KEY porque as linhas-pai se perderam nas paginas corrompidas. O
// -one_at_a_time nao resolve esse caso: ele isola erros nos dados, nao na
// criacao das constraints. A saida tipica do gbak e:
//
//	gbak: ERROR:action cancelled by trigger (2) to preserve data integrity
//	gbak: ERROR:    Non-existent PRIMARY or UNIQUE KEY specified for FOREIGN KEY.
func gbakLooksLikeOrphanForeignKey(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "non-existent primary or unique key specified for foreign key") ||
		(strings.Contains(lower, "action cancelled by trigger") && strings.Contains(lower, "preserve data integrity"))
}

// constraintDefinition descreve uma FOREIGN KEY ou uma CHECK lida das tabelas
// de sistema do banco de origem. O restore com -n (-no_validity) nao recria
// nenhuma das duas, entao a definicao precisa ser extraida e guardada em
// script antes de o -n rodar.
type constraintDefinition struct {
	Name        string
	Relation    string
	Type        string
	Columns     []string
	RefRelation string
	RefColumns  []string
	UpdateRule  string
	DeleteRule  string
	Check       string
	// Truncated marca a CHECK cuja expressao passou do limite do CAST e veio
	// cortada do banco. O DDL dessa constraint nao pode ser executado.
	Truncated bool
}

const (
	constraintTypeForeignKey = "FOREIGN KEY"
	constraintTypeCheck      = "CHECK"

	constraintsScriptName = "CONSTRAINTS_ORIGINAIS.sql"
	constraintsDumpName   = "CONSTRAINTS_ORIGINAIS_SCHEMA_COMPLETO.sql"
	orphanRowsScriptName  = "LINHAS_ORFAS.sql"

	// constraintCheckSourceLimit e o tamanho do CAST que le RDB$TRIGGER_SOURCE.
	// A consulta devolve tambem o CHAR_LENGTH real para que a expressao cortada
	// seja detectada em vez de virar um ALTER TABLE invalido.
	constraintCheckSourceLimit = 4000
)

// constraintsExtractionScript le as FOREIGN KEY e as CHECK pelas tabelas de
// sistema em vez de usar isql -x, que despeja o schema inteiro. A FK sai com
// uma linha por segmento do indice; o Go reagrupa por nome da constraint.
func constraintsExtractionScript() string {
	return "SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\n" +
		"SELECT 'FTB_FKC|' || TRIM(rc.RDB$CONSTRAINT_NAME) || '|' || TRIM(rc.RDB$RELATION_NAME) || '|' || TRIM(s.RDB$FIELD_NAME) || '|' || TRIM(puk.RDB$RELATION_NAME) || '|' || TRIM(ps.RDB$FIELD_NAME) || '|' || TRIM(COALESCE(ref.RDB$UPDATE_RULE, '')) || '|' || TRIM(COALESCE(ref.RDB$DELETE_RULE, '')) " +
		"FROM RDB$RELATION_CONSTRAINTS rc " +
		"JOIN RDB$REF_CONSTRAINTS ref ON ref.RDB$CONSTRAINT_NAME = rc.RDB$CONSTRAINT_NAME " +
		"JOIN RDB$RELATION_CONSTRAINTS puk ON puk.RDB$CONSTRAINT_NAME = ref.RDB$CONST_NAME_UQ " +
		"JOIN RDB$INDEX_SEGMENTS s ON s.RDB$INDEX_NAME = rc.RDB$INDEX_NAME " +
		"JOIN RDB$INDEX_SEGMENTS ps ON ps.RDB$INDEX_NAME = puk.RDB$INDEX_NAME AND ps.RDB$FIELD_POSITION = s.RDB$FIELD_POSITION " +
		"WHERE rc.RDB$CONSTRAINT_TYPE = 'FOREIGN KEY' " +
		"ORDER BY rc.RDB$RELATION_NAME, rc.RDB$CONSTRAINT_NAME, s.RDB$FIELD_POSITION;\n" +
		"SELECT 'FTB_CHK|' || TRIM(rc.RDB$CONSTRAINT_NAME) || '|' || TRIM(rc.RDB$RELATION_NAME) || '|' || CAST(COALESCE(CHAR_LENGTH(t.RDB$TRIGGER_SOURCE), 0) AS VARCHAR(10)) || '|' || TRIM(COALESCE(REPLACE(REPLACE(CAST(t.RDB$TRIGGER_SOURCE AS VARCHAR(" + strconv.Itoa(constraintCheckSourceLimit) + ")), ASCII_CHAR(13), ' '), ASCII_CHAR(10), ' '), '')) " +
		"FROM RDB$RELATION_CONSTRAINTS rc " +
		"JOIN RDB$CHECK_CONSTRAINTS cc ON cc.RDB$CONSTRAINT_NAME = rc.RDB$CONSTRAINT_NAME " +
		"JOIN RDB$TRIGGERS t ON t.RDB$TRIGGER_NAME = cc.RDB$TRIGGER_NAME " +
		"WHERE rc.RDB$CONSTRAINT_TYPE = 'CHECK' AND t.RDB$TRIGGER_TYPE = 1 " +
		"ORDER BY rc.RDB$RELATION_NAME, rc.RDB$CONSTRAINT_NAME;\nQUIT;\n"
}

func parseConstraintDefinitions(output string) []constraintDefinition {
	var order []string
	byKey := map[string]*constraintDefinition{}
	for _, item := range markerLines(output, "FTB_FKC|") {
		parts := strings.Split(item, "|")
		if len(parts) < 5 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		relation := strings.TrimSpace(parts[1])
		column := strings.TrimSpace(parts[2])
		refRelation := strings.TrimSpace(parts[3])
		refColumn := strings.TrimSpace(parts[4])
		if name == "" || relation == "" || column == "" || refRelation == "" || refColumn == "" {
			continue
		}
		key := constraintTypeForeignKey + "|" + name
		current, ok := byKey[key]
		if !ok {
			current = &constraintDefinition{Name: name, Relation: relation, Type: constraintTypeForeignKey, RefRelation: refRelation}
			if len(parts) >= 6 {
				current.UpdateRule = strings.TrimSpace(parts[5])
			}
			if len(parts) >= 7 {
				current.DeleteRule = strings.TrimSpace(parts[6])
			}
			byKey[key] = current
			order = append(order, key)
		}
		current.Columns = append(current.Columns, column)
		current.RefColumns = append(current.RefColumns, refColumn)
	}
	// A CHECK gera um gatilho por evento; a consulta ja filtra o BEFORE INSERT,
	// mas bancos antigos podem devolver mais de uma linha para a mesma
	// constraint. Fica a primeira.
	for _, item := range markerLines(output, "FTB_CHK|") {
		parts := strings.SplitN(item, "|", 4)
		if len(parts) < 4 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		relation := strings.TrimSpace(parts[1])
		source := strings.TrimSpace(parts[3])
		if name == "" || relation == "" || source == "" {
			continue
		}
		key := constraintTypeCheck + "|" + name
		if _, ok := byKey[key]; ok {
			continue
		}
		// O CHAR_LENGTH vem do blob inteiro; o texto vem do CAST limitado. Se o
		// blob for maior que o limite, a expressao chegou cortada. Quando o
		// banco nao sabe informar o tamanho, o proprio texto no limite denuncia.
		fullLength, lengthErr := strconv.Atoi(strings.TrimSpace(parts[2]))
		truncated := fullLength > constraintCheckSourceLimit
		if lengthErr != nil && len([]rune(source)) >= constraintCheckSourceLimit {
			truncated = true
		}
		byKey[key] = &constraintDefinition{Name: name, Relation: relation, Type: constraintTypeCheck, Check: source, Truncated: truncated}
		order = append(order, key)
	}
	result := make([]constraintDefinition, 0, len(order))
	for _, key := range order {
		result = append(result, *byKey[key])
	}
	return result
}

// constraintIdentifiersQuoted decide se o script gerado pode usar aspas duplas.
// No dialeto 1 a aspa dupla e literal de string, nao identificador, e o script
// nao executaria; os nomes de metadados do dialeto 1 sao sempre maiusculos e
// sem espaco, entao o nome cru serve. Dialeto desconhecido cai no 3, que e o
// caso comum e o unico em que um nome com minuscula ou espaco pode existir.
func constraintIdentifiersQuoted(dialect string) bool {
	return strings.TrimSpace(dialect) != "1"
}

// constraintIdentifier e o sqlIdentifier do script de constraints. O
// sqlIdentifier original e usado em outros pontos e continua intocado.
func constraintIdentifier(name string, quoted bool) string {
	if quoted {
		return sqlIdentifier(name)
	}
	return strings.TrimSpace(name)
}

func constraintIdentifierList(columns []string, quoted bool) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, constraintIdentifier(column, quoted))
	}
	return strings.Join(parts, ", ")
}

// foreignKeyRuleClause omite RESTRICT e NO ACTION, que sao o comportamento
// padrao e nao precisam aparecer no DDL.
func foreignKeyRuleClause(event, rule string) string {
	rule = strings.ToUpper(strings.TrimSpace(rule))
	if rule == "" || rule == "RESTRICT" || rule == "NO ACTION" {
		return ""
	}
	return " ON " + event + " " + rule
}

func constraintDDL(c constraintDefinition, quoted bool) string {
	switch c.Type {
	case constraintTypeForeignKey:
		if c.Name == "" || c.Relation == "" || c.RefRelation == "" || len(c.Columns) == 0 || len(c.Columns) != len(c.RefColumns) {
			return ""
		}
		return "ALTER TABLE " + constraintIdentifier(c.Relation, quoted) + " ADD CONSTRAINT " + constraintIdentifier(c.Name, quoted) +
			" FOREIGN KEY (" + constraintIdentifierList(c.Columns, quoted) + ") REFERENCES " + constraintIdentifier(c.RefRelation, quoted) +
			" (" + constraintIdentifierList(c.RefColumns, quoted) + ")" +
			foreignKeyRuleClause("UPDATE", c.UpdateRule) + foreignKeyRuleClause("DELETE", c.DeleteRule) + ";"
	case constraintTypeCheck:
		expr := strings.TrimSpace(c.Check)
		if c.Name == "" || c.Relation == "" || expr == "" {
			return ""
		}
		if !strings.HasPrefix(strings.ToUpper(expr), "CHECK") {
			expr = "CHECK " + expr
		}
		return "ALTER TABLE " + constraintIdentifier(c.Relation, quoted) + " ADD CONSTRAINT " + constraintIdentifier(c.Name, quoted) + " " + expr + ";"
	}
	return ""
}

// truncatedCheckConstraints lista as CHECK cuja expressao veio cortada do
// banco. Elas nao podem ser recriadas pelo script e precisam ser recuperadas
// a mao.
func truncatedCheckConstraints(constraints []constraintDefinition) []string {
	var names []string
	for _, c := range constraints {
		if c.Type == constraintTypeCheck && c.Truncated {
			names = append(names, c.Name+" em "+c.Relation)
		}
	}
	return names
}

func buildConstraintsDDLScript(constraints []constraintDefinition, dialect string) string {
	quoted := constraintIdentifiersQuoted(dialect)
	var b strings.Builder
	b.WriteString("/* FIREBIRD TOOLBOX - CONSTRAINTS ORIGINAIS DO BANCO DE ORIGEM */\n")
	b.WriteString("/* O banco restaurado com -n (-no_validity) NAO possui estas constraints. */\n")
	b.WriteString("/* Sanear as linhas orfas (ver " + orphanRowsScriptName + ") ANTES de executar este script. */\n")
	if !quoted {
		b.WriteString("/* Banco em dialeto 1: os identificadores vao sem aspas duplas, que ali sao literal de string. */\n")
	}
	b.WriteString("/* Execucao: isql -user SYSDBA -password <senha> <banco> -i " + constraintsScriptName + " */\n\n")
	if len(constraints) == 0 {
		b.WriteString("/* Nenhuma FOREIGN KEY ou CHECK foi lida das tabelas de sistema do banco de origem. */\n")
		return b.String()
	}
	b.WriteString("SET AUTODDL OFF;\n\n")
	written := 0
	for _, c := range constraints {
		ddl := constraintDDL(c, quoted)
		if ddl == "" {
			continue
		}
		// A expressao cortada geraria um ALTER TABLE invalido. Sai comentada,
		// com o aviso de que precisa ser recuperada a mao.
		if c.Truncated {
			b.WriteString("-- ATENCAO: a expressao desta CHECK passa de " + strconv.Itoa(constraintCheckSourceLimit) + " caracteres e foi lida CORTADA.\n")
			b.WriteString("-- O comando abaixo esta INCOMPLETO e fica comentado. Recupere a expressao original antes de executar.\n")
			for _, line := range strings.Split(ddl, "\n") {
				b.WriteString("-- " + line + "\n")
			}
			b.WriteString("\n")
			continue
		}
		b.WriteString("/* " + c.Type + " " + c.Name + " em " + c.Relation + " */\n")
		b.WriteString(ddl + "\n")
		b.WriteString("COMMIT;\n\n")
		written++
	}
	if written == 0 {
		b.WriteString("/* Nenhuma constraint pode ser reconstruida com os metadados lidos. */\n")
	}
	return b.String()
}

// orphanRowsQuery monta a consulta que lista os filhos sem pai para uma FK.
// So consulta: a decisao de apagar ou corrigir e do tecnico.
func orphanRowsQuery(c constraintDefinition, quoted bool) string {
	if c.Type != constraintTypeForeignKey || c.Relation == "" || c.RefRelation == "" {
		return ""
	}
	if len(c.Columns) == 0 || len(c.Columns) != len(c.RefColumns) {
		return ""
	}
	notNull := make([]string, 0, len(c.Columns))
	match := make([]string, 0, len(c.Columns))
	for i, column := range c.Columns {
		notNull = append(notNull, "F."+constraintIdentifier(column, quoted)+" IS NOT NULL")
		match = append(match, "P."+constraintIdentifier(c.RefColumns[i], quoted)+" = F."+constraintIdentifier(column, quoted))
	}
	where := strings.Join(notNull, " AND ") +
		" AND NOT EXISTS (SELECT 1 FROM " + constraintIdentifier(c.RefRelation, quoted) + " P WHERE " + strings.Join(match, " AND ") + ")"
	return "SELECT COUNT(*) FROM " + constraintIdentifier(c.Relation, quoted) + " F WHERE " + where + ";\n" +
		"SELECT F.* FROM " + constraintIdentifier(c.Relation, quoted) + " F WHERE " + where + ";\n\n"
}

func buildOrphanRowsScript(constraints []constraintDefinition, dialect string) string {
	quoted := constraintIdentifiersQuoted(dialect)
	var b strings.Builder
	b.WriteString("/* FIREBIRD TOOLBOX - LINHAS ORFAS POR FOREIGN KEY */\n")
	b.WriteString("/* Somente consultas: este script NAO apaga e NAO altera nenhuma linha. */\n")
	b.WriteString("/* Inspecione cada resultado antes de decidir apagar ou corrigir as linhas filhas. */\n")
	if !quoted {
		b.WriteString("/* Banco em dialeto 1: os identificadores vao sem aspas duplas, que ali sao literal de string. */\n")
	}
	b.WriteString("/* Depois de sanear, recrie as constraints com " + constraintsScriptName + ". */\n\n")
	written := 0
	for _, c := range constraints {
		query := orphanRowsQuery(c, quoted)
		if query == "" {
			continue
		}
		b.WriteString("/* " + c.Name + ": " + c.Relation + " -> " + c.RefRelation + " */\n")
		b.WriteString(query)
		written++
	}
	if written == 0 {
		b.WriteString("/* Nenhuma FOREIGN KEY foi lida do banco de origem. */\n")
	}
	return b.String()
}

// databaseDialectOf le o dialeto pelo cabecalho do gstat. Volta vazio quando
// nao da para saber, e ai o script e gerado como dialeto 3.
func databaseDialectOf(ctx context.Context, root, version, dbPath string) string {
	gstat := firebirdExe(root, version, "gstat.exe")
	if !fileExists(gstat) {
		return ""
	}
	output, err := runCommandCapture(ctx, gstat, "-h", dbPath)
	if err != nil {
		return ""
	}
	return parseDatabaseHeader(output).Dialect
}

// constraintSourceCandidates lista os bancos de onde o DDL pode ser lido, da
// copia de trabalho (ja saneada) para o original.
func constraintSourceCandidates(c workflowConfig, report *recoveryReport) []string {
	var candidates []string
	if report != nil && report.WorkingCopy != "" && fileExists(report.WorkingCopy) {
		candidates = append(candidates, report.WorkingCopy)
	}
	if c.Source != "" && fileExists(c.Source) {
		candidates = append(candidates, c.Source)
	}
	return candidates
}

// preserveConstraintsBeforeNoValidity grava o DDL das constraints na pasta
// RECOVERY_* antes de o restore com -n apaga-las dos metadados. Nunca aborta a
// recuperacao: um restore sem constraints continua melhor que nenhum restore,
// entao toda falha vira aviso no relatorio.
func preserveConstraintsBeforeNoValidity(ctx context.Context, c workflowConfig, report *recoveryReport, dir string) []constraintDefinition {
	if report == nil || dir == "" {
		return nil
	}
	candidates := constraintSourceCandidates(c, report)
	if len(candidates) == 0 {
		report.warning("Não havia banco de origem acessível para extrair o DDL das constraints antes do restore com -n")
		return nil
	}
	var lastErr error
	for _, dbPath := range candidates {
		out, err := runISQLScript(ctx, c.FbSource, c.SourceVersion, c.SourcePort, c.User, c.Password, dbPath, "Extraindo o DDL das constraints antes do restore com -n...", constraintsExtractionScript())
		constraints := parseConstraintDefinitions(out)
		if len(constraints) == 0 {
			if err != nil {
				lastErr = err
			}
			continue
		}
		if err != nil {
			report.warning("A leitura das constraints retornou erro parcial; o script pode estar incompleto: " + err.Error())
		}
		report.ConstraintsDialect = databaseDialectOf(ctx, c.FbSource, c.SourceVersion, dbPath)
		if !constraintIdentifiersQuoted(report.ConstraintsDialect) {
			report.action("Banco de origem em dialeto 1: o script das constraints foi gerado sem aspas duplas nos identificadores")
		}
		for _, name := range truncatedCheckConstraints(constraints) {
			report.warning("A expressão da CHECK " + name + " passa de " + strconv.Itoa(constraintCheckSourceLimit) + " caracteres e foi lida cortada; ela sai comentada em " + constraintsScriptName + " e precisa ser recuperada manualmente")
		}
		path := filepath.Join(dir, constraintsScriptName)
		if writeErr := os.WriteFile(path, []byte(buildConstraintsDDLScript(constraints, report.ConstraintsDialect)), 0600); writeErr != nil {
			report.warning("Não foi possível gravar " + constraintsScriptName + ": " + writeErr.Error())
			return constraints
		}
		report.ConstraintsScript = path
		report.ConstraintsExtracted = len(constraints)
		report.action(fmt.Sprintf("DDL de %d constraint(s) extraído de %s e preservado em %s antes do restore com -n", len(constraints), dbPath, constraintsScriptName))
		appendLog("✓ DDL das constraints preservado em: " + path)
		return constraints
	}
	detail := ""
	if lastErr != nil {
		detail = ": " + lastErr.Error()
	}
	report.warning("Não foi possível ler as constraints pelas tabelas de sistema antes do restore com -n" + detail)
	dumpSchemaBeforeNoValidity(ctx, c, report, dir, candidates)
	return nil
}

// dumpSchemaBeforeNoValidity e a alternativa quando a consulta as tabelas de
// sistema falha: isql -x despeja o schema inteiro, de onde o tecnico extrai os
// ALTER TABLE ... ADD CONSTRAINT a mao.
func dumpSchemaBeforeNoValidity(ctx context.Context, c workflowConfig, report *recoveryReport, dir string, candidates []string) {
	isql := firebirdExe(c.FbSource, c.SourceVersion, "isql.exe")
	if !fileExists(isql) {
		report.warning("Não encontrei isql.exe do Firebird " + c.SourceVersion + " para extrair o schema completo como alternativa")
		return
	}
	path := filepath.Join(dir, constraintsDumpName)
	for _, dbPath := range candidates {
		_ = os.Remove(path)
		// O -o manda a saida direto para o arquivo: o schema completo nao cabe
		// no limite de linhas capturadas pelo log.
		_, err := runRecoveryCapture(ctx, "Extraindo o schema completo com isql -x (alternativa)...", isql,
			"-x", "-o", path, "-user", c.User, "-password", c.Password, databaseConnection(c.SourcePort, dbPath))
		if err != nil {
			continue
		}
		if info, statErr := os.Stat(path); statErr != nil || info.Size() == 0 {
			continue
		}
		report.ConstraintsDump = path
		report.action("Schema completo extraído com isql -x de " + dbPath + " e preservado em " + constraintsDumpName)
		appendLog("✓ Schema completo preservado em: " + path)
		return
	}
	_ = os.Remove(path)
	report.warning("Também não foi possível extrair o schema completo com isql -x; a definição das constraints terá de vir de um backup anterior")
}

// writeOrphanRowsInspectionScript grava as consultas que listam os filhos sem
// pai, para o tecnico inspecionar antes de apagar qualquer coisa.
func writeOrphanRowsInspectionScript(report *recoveryReport, dir string, constraints []constraintDefinition) {
	if report == nil || dir == "" {
		return
	}
	foreignKeys := 0
	for _, c := range constraints {
		if c.Type == constraintTypeForeignKey {
			foreignKeys++
		}
	}
	if foreignKeys == 0 {
		report.warning("Nenhuma FOREIGN KEY foi lida do banco de origem; o script de inspeção de linhas órfãs não foi gerado")
		return
	}
	path := filepath.Join(dir, orphanRowsScriptName)
	if err := os.WriteFile(path, []byte(buildOrphanRowsScript(constraints, report.ConstraintsDialect)), 0600); err != nil {
		report.warning("Não foi possível gravar " + orphanRowsScriptName + ": " + err.Error())
		return
	}
	report.OrphanRowsScript = path
	report.action(fmt.Sprintf("Script de inspeção de linhas órfãs gerado para %d FOREIGN KEY", foreignKeys))
	appendLog("✓ Consultas de linhas órfãs preservadas em: " + path)
}

func metadataBackupPath(backup string) string {
	ext := filepath.Ext(backup)
	base := strings.TrimSuffix(backup, ext)
	if ext == "" {
		ext = ".BAK"
	}
	return base + "_METADATA" + ext
}

func metadataBackupArgs(c workflowConfig, dbConn, backup string) []string {
	return []string{"-b", "-v", "-m", "-user", c.User, "-pass", c.Password, dbConn, backup}
}

type relationColumn struct {
	Name         string
	HasArray     bool
	Generated    bool
	FieldType    int
	FieldSubType int
}

type relationMetadata struct {
	Name              string
	Columns           []relationColumn
	PrimaryKey        string
	PrimaryIndex      string
	PrimaryType       int
	PrimaryScale      int
	ActiveIndexes     []string
	ConstraintIndexes []string
	ActiveTriggers    []string
}

type rowRescueResult struct {
	Attempted bool
	Complete  bool
	Recovered int64
	LostKeys  []string
	StagePath string
	Backup    string
}

func sqlStringLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func sqlIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func writeRecoveryTechLine(text string) {
	path := recoveryTechnicalLogPath()
	if path == "" || strings.TrimSpace(text) == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	stamp := time.Now().Format("15:04:05")
	for _, line := range strings.Split(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\n") {
		_, _ = fmt.Fprintf(f, "[%s] %s\r\n", stamp, line)
	}
}

func runISQLScript(ctx context.Context, root, version, port, user, password, dbPath, label, script string) (string, error) {
	isql := firebirdExe(root, version, "isql.exe")
	if !fileExists(isql) {
		return "", fmt.Errorf("não encontrei isql.exe do Firebird %s", version)
	}
	dbConn := databaseConnection(port, dbPath)
	if detailedLog {
		appendLog(label)
	}
	writeRecoveryTechLine(label)
	writeRecoveryTechLine("ISQL: " + quoteForLog(isql) + " -user " + user + " -password ******** " + dbConn)

	cmd := exec.CommandContext(ctx, isql, "-user", user, "-password", password, dbConn)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}
	cmd.Stdin = strings.NewReader(script)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	text := strings.TrimSpace(decodeWindowsBytes(out.Bytes()))
	if err == nil && isqlOutputHasFailure(text) {
		err = errors.New("isql reportou falha na execução do SQL")
	}
	if text != "" {
		writeRecoveryTechLine(text)
		if detailedLog {
			appendLog(text)
		}
	}
	if err != nil {
		writeRecoveryTechLine("ISQL FINALIZADO COM ERRO: " + err.Error())
	} else {
		writeRecoveryTechLine("ISQL FINALIZADO COM SUCESSO")
	}
	if ctx.Err() != nil {
		return text, ctx.Err()
	}
	return text, err
}

func isqlOutputHasFailure(output string) bool {
	lower := strings.ToLower(output)
	for _, token := range []string{
		"statement failed, sqlstate",
		"dynamic sql error",
		"sql error code",
		"unsuccessful metadata update",
		"attempt to store duplicate value",
		"validation error for column",
		"violation of primary or unique key constraint",
		"violation of foreign key constraint",
		"database file appears corrupt",
		"wrong page type",
		"page is of wrong type",
		"error: unable to complete network request",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func markerLines(output, prefix string) []string {
	var result []string
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if idx := strings.Index(line, prefix); idx >= 0 {
			result = append(result, strings.TrimSpace(line[idx+len(prefix):]))
		}
	}
	return result
}

func relationMetadataScript(relation string) string {
	rel := sqlStringLiteral(relation)
	return "SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\n" +
		"SELECT 'FTB_COL|' || TRIM(rf.RDB$FIELD_NAME) || '|' || CAST(COALESCE(f.RDB$DIMENSIONS, 0) AS VARCHAR(10)) || '|' || CASE WHEN rf.RDB$GENERATOR_NAME IS NULL THEN '0' ELSE '1' END || '|' || CAST(f.RDB$FIELD_TYPE AS VARCHAR(10)) || '|' || CAST(COALESCE(f.RDB$FIELD_SUB_TYPE, 0) AS VARCHAR(10)) " +
		"FROM RDB$RELATION_FIELDS rf JOIN RDB$FIELDS f ON f.RDB$FIELD_NAME = rf.RDB$FIELD_SOURCE " +
		"WHERE rf.RDB$RELATION_NAME = " + rel + " AND COALESCE(rf.RDB$UPDATE_FLAG, 1) = 1 AND f.RDB$COMPUTED_BLR IS NULL ORDER BY rf.RDB$FIELD_POSITION;\n" +
		"SELECT 'FTB_PK|' || TRIM(s.RDB$FIELD_NAME) || '|' || TRIM(rc.RDB$INDEX_NAME) || '|' || CAST(f.RDB$FIELD_TYPE AS VARCHAR(10)) || '|' || CAST(COALESCE(f.RDB$FIELD_SCALE,0) AS VARCHAR(10)) " +
		"FROM RDB$RELATION_CONSTRAINTS rc JOIN RDB$INDEX_SEGMENTS s ON s.RDB$INDEX_NAME = rc.RDB$INDEX_NAME " +
		"JOIN RDB$RELATION_FIELDS rf ON rf.RDB$RELATION_NAME = rc.RDB$RELATION_NAME AND rf.RDB$FIELD_NAME = s.RDB$FIELD_NAME " +
		"JOIN RDB$FIELDS f ON f.RDB$FIELD_NAME = rf.RDB$FIELD_SOURCE WHERE rc.RDB$RELATION_NAME = " + rel + " AND rc.RDB$CONSTRAINT_TYPE = 'PRIMARY KEY' ORDER BY s.RDB$FIELD_POSITION;\n" +
		"SELECT 'FTB_IDX|' || TRIM(i.RDB$INDEX_NAME) FROM RDB$INDICES i WHERE i.RDB$RELATION_NAME = " + rel + " AND COALESCE(i.RDB$SYSTEM_FLAG,0) = 0 AND COALESCE(i.RDB$INDEX_INACTIVE,0) = 0 ORDER BY i.RDB$INDEX_NAME;\n" +
		"SELECT 'FTB_CIDX|' || TRIM(rc.RDB$INDEX_NAME) FROM RDB$RELATION_CONSTRAINTS rc WHERE rc.RDB$RELATION_NAME = " + rel + " AND rc.RDB$INDEX_NAME IS NOT NULL ORDER BY rc.RDB$INDEX_NAME;\n" +
		"SELECT 'FTB_TRG|' || TRIM(t.RDB$TRIGGER_NAME) FROM RDB$TRIGGERS t WHERE t.RDB$RELATION_NAME = " + rel + " AND COALESCE(t.RDB$SYSTEM_FLAG,0) = 0 AND COALESCE(t.RDB$TRIGGER_INACTIVE,0) = 0 ORDER BY t.RDB$TRIGGER_NAME;\nQUIT;\n"
}

func loadRelationMetadataAt(ctx context.Context, root, version, port, user, password, dbPath, relation string) (relationMetadata, error) {
	out, err := runISQLScript(ctx, root, version, port, user, password, dbPath, "Lendo metadados da relação "+relation+"...", relationMetadataScript(relation))
	if err != nil {
		return relationMetadata{}, err
	}
	meta := relationMetadata{Name: relation}
	for _, item := range markerLines(out, "FTB_COL|") {
		parts := strings.Split(item, "|")
		if len(parts) < 3 {
			continue
		}
		fieldType, fieldSubType := 0, 0
		if len(parts) >= 4 {
			fieldType, _ = strconv.Atoi(strings.TrimSpace(parts[3]))
		}
		if len(parts) >= 5 {
			fieldSubType, _ = strconv.Atoi(strings.TrimSpace(parts[4]))
		}
		meta.Columns = append(meta.Columns, relationColumn{
			Name: strings.TrimSpace(parts[0]), HasArray: strings.TrimSpace(parts[1]) != "0", Generated: strings.TrimSpace(parts[2]) == "1",
			FieldType: fieldType, FieldSubType: fieldSubType,
		})
	}
	pkLines := markerLines(out, "FTB_PK|")
	if len(pkLines) == 1 {
		parts := strings.Split(pkLines[0], "|")
		if len(parts) >= 4 {
			meta.PrimaryKey = strings.TrimSpace(parts[0])
			meta.PrimaryIndex = strings.TrimSpace(parts[1])
			meta.PrimaryType, _ = strconv.Atoi(strings.TrimSpace(parts[2]))
			meta.PrimaryScale, _ = strconv.Atoi(strings.TrimSpace(parts[3]))
		}
	}
	meta.ActiveIndexes = markerLines(out, "FTB_IDX|")
	meta.ConstraintIndexes = markerLines(out, "FTB_CIDX|")
	meta.ActiveTriggers = markerLines(out, "FTB_TRG|")
	if len(meta.Columns) == 0 {
		return meta, fmt.Errorf("não foi possível obter as colunas da relação %s", relation)
	}
	return meta, nil
}

func loadRelationMetadata(ctx context.Context, c workflowConfig, port, dbPath, relation string) (relationMetadata, error) {
	return loadRelationMetadataAt(ctx, c.FbSource, c.SourceVersion, port, c.User, c.Password, dbPath, relation)
}

func integerPrimaryKey(meta relationMetadata) bool {
	if meta.PrimaryKey == "" || meta.PrimaryScale != 0 {
		return false
	}
	switch meta.PrimaryType {
	case 7, 8, 16, 26:
		return true
	default:
		return false
	}
}

func hasUnsupportedArray(meta relationMetadata) bool {
	for _, col := range meta.Columns {
		if col.HasArray {
			return true
		}
	}
	return false
}

func buildAlterIndexesScript(indexes []string, active bool) string {
	if len(indexes) == 0 {
		return "QUIT;\n"
	}
	state := "INACTIVE"
	if active {
		state = "ACTIVE"
	}
	var b strings.Builder
	b.WriteString("SET BAIL OFF;\n")
	for _, index := range indexes {
		b.WriteString("ALTER INDEX " + sqlIdentifier(index) + " " + state + ";\nCOMMIT;\n")
	}
	b.WriteString("QUIT;\n")
	return b.String()
}

func deactivateRelationIndexes(ctx context.Context, c workflowConfig, port, dbPath string, meta relationMetadata) []string {
	constraint := make(map[string]bool, len(meta.ConstraintIndexes))
	for _, index := range meta.ConstraintIndexes {
		constraint[strings.ToUpper(strings.TrimSpace(index))] = true
	}
	var disabled []string
	for _, index := range meta.ActiveIndexes {
		if constraint[strings.ToUpper(strings.TrimSpace(index))] {
			continue
		}
		script := "SET BAIL ON;\nALTER INDEX " + sqlIdentifier(index) + " INACTIVE;\nCOMMIT;\nQUIT;\n"
		if _, err := runISQLScript(ctx, c.FbSource, c.SourceVersion, port, c.User, c.Password, dbPath, "Tentando desativar índice "+index+" somente na cópia...", script); err == nil {
			disabled = append(disabled, index)
		}
	}
	return disabled
}

func salvageEngineConfig(c workflowConfig) (version, root, port string) {
	// O banco temporário de resgate sempre usa a mesma engine da origem.
	// Isso evita que uma migração regressiva interfira no salvamento dos dados:
	// primeiro recuperamos o máximo possível na versão que consegue abrir o banco
	// danificado e somente depois testamos/restauramos na versão de destino.
	return c.SourceVersion, c.FbSource, c.SourcePort
}

func runStageISQL(ctx context.Context, c workflowConfig, dbPath, label, script string) (string, error) {
	version, root, port := salvageEngineConfig(c)
	resolved := port
	if supportsServiceManagerRestore(version) {
		p, err := verifyFirebirdEndpoint(ctx, "do resgate de registros", version, root, "", port)
		if err != nil {
			return "", err
		}
		resolved = p
	}
	return runISQLScript(ctx, root, version, resolved, c.User, c.Password, dbPath, label, script)
}

func restoreBackupToPath(ctx context.Context, c workflowConfig, backup, dest, label string) error {
	version, root, port := salvageEngineConfig(c)
	gbak := firebirdExe(root, version, "gbak.exe")
	if !fileExists(gbak) {
		return fmt.Errorf("não encontrei gbak.exe do Firebird %s", version)
	}
	resolved := port
	if supportsServiceManagerRestore(version) {
		p, err := verifyFirebirdEndpoint(ctx, "do resgate de registros", version, root, "", port)
		if err != nil {
			return err
		}
		resolved = p
	}
	_ = os.Remove(dest)
	_, err := runRecoveryCapture(ctx, label, gbak, restoreTestArgs(version, resolved, c.User, c.Password, backup, dest, false, false)...)
	if err != nil {
		return err
	}
	if !fileExists(dest) {
		return fmt.Errorf("a restauração de apoio terminou sem criar %s", dest)
	}
	return nil
}

func backupDatabasePath(ctx context.Context, c workflowConfig, dbPath, backup, label string) error {
	version, root, port := salvageEngineConfig(c)
	gbak := firebirdExe(root, version, "gbak.exe")
	if !fileExists(gbak) {
		return fmt.Errorf("não encontrei gbak.exe do Firebird %s", version)
	}
	resolved := port
	if supportsServiceManagerRestore(version) {
		p, err := verifyFirebirdEndpoint(ctx, "do backup pós-resgate", version, root, "", port)
		if err != nil {
			return err
		}
		resolved = p
	}
	_ = os.Remove(backup)
	dbConn := databaseConnection(resolved, dbPath)
	_, err := runRecoveryCapture(ctx, label, gbak, "-b", "-v", "-user", c.User, "-pass", c.Password, dbConn, backup)
	if err != nil {
		return err
	}
	if !fileExists(backup) {
		return fmt.Errorf("o backup pós-resgate não foi criado: %s", backup)
	}
	return nil
}

func stagingPathForRecovery(backup, relation string) string {
	ext := filepath.Ext(backup)
	base := strings.TrimSuffix(backup, ext)
	return base + "_SALVAGE_" + sanitizeFilePart(relation) + ".FDB"
}

func sanitizeFilePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "TABLE"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func recoveredBackupPath(backup string) string {
	ext := filepath.Ext(backup)
	base := strings.TrimSuffix(backup, ext)
	if ext == "" {
		ext = ".BAK"
	}
	return base + "_COM_ITENS" + ext
}

func basePartialBackupPath(backup, relation string) string {
	ext := filepath.Ext(backup)
	base := strings.TrimSuffix(backup, ext)
	if ext == "" {
		ext = ".BAK"
	}
	return base + "_SEM_" + sanitizeFilePart(relation) + ext
}

func externalDataSourceAt(root, version, port, stagePath string) string {
	if !portReachable(port) {
		if detected := detectFirebirdPort(root, version); detected != "" {
			port = detected
		}
	}
	if strings.TrimSpace(port) == "" || port == "3050" {
		return "127.0.0.1:" + stagePath
	}
	return "127.0.0.1/" + port + ":" + stagePath
}

func externalDataSource(c workflowConfig, stagePath string) string {
	version, root, port := salvageEngineConfig(c)
	return externalDataSourceAt(root, version, port, stagePath)
}

func copyRelationBlockToDataSource(meta relationMetadata, user, password, dataSource, whereClause string, natural bool) (string, error) {
	if len(meta.Columns) == 0 {
		return "", errors.New("relação sem colunas copiáveis")
	}
	if hasUnsupportedArray(meta) {
		return "", errors.New("a relação possui coluna ARRAY; o resgate automático de registros não suporta ARRAY")
	}
	var decls, vars, cols, placeholders []string
	for i, col := range meta.Columns {
		v := fmt.Sprintf("V%d", i+1)
		decls = append(decls, "DECLARE VARIABLE "+v+" TYPE OF COLUMN "+sqlIdentifier(meta.Name)+"."+sqlIdentifier(col.Name)+";")
		vars = append(vars, ":"+v)
		cols = append(cols, sqlIdentifier(col.Name))
		placeholders = append(placeholders, "?")
	}
	insertSQL := "INSERT INTO " + sqlIdentifier(meta.Name) + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	if len(insertSQL) > 30000 {
		return "", errors.New("a instrução de cópia excede o limite seguro do EXECUTE STATEMENT")
	}
	selectSQL := "SELECT " + strings.Join(cols, ", ") + " FROM " + sqlIdentifier(meta.Name) + " T"
	if strings.TrimSpace(whereClause) != "" {
		selectSQL += " WHERE " + whereClause
	}
	if natural {
		selectSQL += " PLAN (T NATURAL)"
	}

	var b strings.Builder
	b.WriteString("SET BAIL ON;\nSET TERM ^;\nEXECUTE BLOCK RETURNS (FTB_RESULT VARCHAR(80)) AS\n")
	for _, decl := range decls {
		b.WriteString(decl + "\n")
	}
	b.WriteString("DECLARE VARIABLE N BIGINT;\nBEGIN\n  N = 0;\n")
	b.WriteString("  FOR " + selectSQL + " INTO " + strings.Join(vars, ", ") + " DO\n  BEGIN\n")
	b.WriteString("    EXECUTE STATEMENT (" + sqlStringLiteral(insertSQL) + ") (" + strings.Join(vars, ", ") + ")\n")
	b.WriteString("      ON EXTERNAL DATA SOURCE " + sqlStringLiteral(dataSource) + " AS USER " + sqlStringLiteral(user) + " PASSWORD " + sqlStringLiteral(password) + " WITH COMMON TRANSACTION;\n")
	b.WriteString("    N = N + 1;\n  END\n  FTB_RESULT = 'FTB_ROWS=' || CAST(N AS VARCHAR(30));\n  SUSPEND;\nEND^\nSET TERM ;^\nCOMMIT;\nQUIT;\n")
	return b.String(), nil
}

func copyRelationBlock(meta relationMetadata, c workflowConfig, stagePath string, whereClause string, natural bool) (string, error) {
	return copyRelationBlockToDataSource(meta, c.User, c.Password, externalDataSource(c, stagePath), whereClause, natural)
}

func parseRecoveredRows(output string) int64 {
	re := regexp.MustCompile(`(?i)FTB_ROWS\s*=\s*(\d+)`)
	m := re.FindStringSubmatch(output)
	if len(m) != 2 {
		return 0
	}
	n, _ := strconv.ParseInt(m[1], 10, 64)
	return n
}

func clearRelationOnStage(ctx context.Context, c workflowConfig, relation string, stagePath string) error {
	script := "SET BAIL ON;\nDELETE FROM " + sqlIdentifier(relation) + ";\nCOMMIT;\nQUIT;\n"
	_, err := runStageISQL(ctx, c, stagePath, "Limpando a relação temporária antes de nova tentativa...", script)
	return err
}

func disableStageTriggers(ctx context.Context, c workflowConfig, stagePath string, triggers []string) error {
	if len(triggers) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("SET BAIL ON;\n")
	for _, name := range triggers {
		b.WriteString("ALTER TRIGGER " + sqlIdentifier(name) + " INACTIVE;\n")
	}
	b.WriteString("COMMIT;\nQUIT;\n")
	_, err := runStageISQL(ctx, c, stagePath, "Desativando temporariamente triggers da relação no banco de resgate...", b.String())
	return err
}

func enableStageTriggers(ctx context.Context, c workflowConfig, stagePath string, triggers []string) error {
	if len(triggers) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("SET BAIL ON;\n")
	for _, name := range triggers {
		b.WriteString("ALTER TRIGGER " + sqlIdentifier(name) + " ACTIVE;\n")
	}
	b.WriteString("COMMIT;\nQUIT;\n")
	_, err := runStageISQL(ctx, c, stagePath, "Reativando triggers da relação no banco de resgate...", b.String())
	return err
}

func primaryKeyBounds(ctx context.Context, c workflowConfig, port, workingPath string, meta relationMetadata) (int64, int64, error) {
	if !integerPrimaryKey(meta) {
		return 0, 0, errors.New("a relação não possui chave primária numérica simples para isolamento por faixas")
	}
	pk := sqlIdentifier(meta.PrimaryKey)
	script := "SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\nSELECT 'FTB_MIN=' || CAST(MIN(" + pk + ") AS VARCHAR(40)) || '|FTB_MAX=' || CAST(MAX(" + pk + ") AS VARCHAR(40)) FROM " + sqlIdentifier(meta.Name) + ";\nQUIT;\n"
	out, err := runISQLScript(ctx, c.FbSource, c.SourceVersion, port, c.User, c.Password, workingPath, "Determinando faixa da chave primária para resgate seletivo...", script)
	if err != nil {
		return 0, 0, err
	}
	re := regexp.MustCompile(`FTB_MIN=\s*(-?\d+)\|FTB_MAX=\s*(-?\d+)`)
	m := re.FindStringSubmatch(out)
	if len(m) != 3 {
		return 0, 0, errors.New("não foi possível determinar os limites da chave primária")
	}
	minValue, err1 := strconv.ParseInt(strings.TrimSpace(m[1]), 10, 64)
	maxValue, err2 := strconv.ParseInt(strings.TrimSpace(m[2]), 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, errors.New("limites da chave primária inválidos")
	}
	return minValue, maxValue, nil
}

func copyRelationRange(ctx context.Context, c workflowConfig, port, workingPath, stagePath string, meta relationMetadata, low, high int64) (int64, error) {
	where := fmt.Sprintf("%s BETWEEN %d AND %d", sqlIdentifier(meta.PrimaryKey), low, high)
	script, err := copyRelationBlock(meta, c, stagePath, where, false)
	if err != nil {
		return 0, err
	}
	out, err := runISQLScript(ctx, c.FbSource, c.SourceVersion, port, c.User, c.Password, workingPath, fmt.Sprintf("Resgatando %s pela chave %d..%d...", meta.Name, low, high), script)
	if err != nil {
		return 0, err
	}
	return parseRecoveredRows(out), nil
}

func rescueRanges(ctx context.Context, c workflowConfig, port, workingPath, stagePath string, meta relationMetadata, low, high int64, attempts *int, maxAttempts int) (int64, []string, error) {
	if low > high {
		return 0, nil, nil
	}
	if *attempts >= maxAttempts {
		return 0, nil, fmt.Errorf("limite de %d tentativas atingido no isolamento de registros", maxAttempts)
	}
	(*attempts)++
	rows, err := copyRelationRange(ctx, c, port, workingPath, stagePath, meta, low, high)
	if err == nil {
		return rows, nil, nil
	}
	if low == high {
		writeRecoveryTechLine(fmt.Sprintf("CHAVE NÃO RECUPERADA: %s=%d | %v", meta.PrimaryKey, low, err))
		return 0, []string{fmt.Sprintf("%s=%d", meta.PrimaryKey, low)}, nil
	}
	mid := low + (high-low)/2
	leftRows, leftLost, leftErr := rescueRanges(ctx, c, port, workingPath, stagePath, meta, low, mid, attempts, maxAttempts)
	if leftErr != nil {
		return leftRows, leftLost, leftErr
	}
	rightRows, rightLost, rightErr := rescueRanges(ctx, c, port, workingPath, stagePath, meta, mid+1, high, attempts, maxAttempts)
	return leftRows + rightRows, append(leftLost, rightLost...), rightErr
}

func attemptRowRescue(ctx context.Context, c workflowConfig, sourcePort, workingPath, baseBackup, finalBackup, relation string, indexesToReactivate []string) (rowRescueResult, error) {
	result := rowRescueResult{Attempted: true}
	if versionNumber(c.SourceVersion) < 2.5 {
		return result, errors.New("resgate de registros exige Firebird 2.5 ou superior por usar conexão externa parametrizada")
	}
	meta, err := loadRelationMetadata(ctx, c, sourcePort, workingPath, relation)
	if err != nil {
		return result, err
	}
	if hasUnsupportedArray(meta) {
		return result, errors.New("a relação possui ARRAY e não pode ser copiada automaticamente com segurança")
	}
	stagePath := stagingPathForRecovery(baseBackup, relation)
	result.StagePath = stagePath
	if err := restoreBackupToPath(ctx, c, baseBackup, stagePath, "Criando banco temporário para recuperar os registros de "+relation+"..."); err != nil {
		return result, fmt.Errorf("não foi possível criar banco temporário de resgate: %w", err)
	}
	if err := disableStageTriggers(ctx, c, stagePath, meta.ActiveTriggers); err != nil {
		return result, fmt.Errorf("não foi possível desativar os triggers da relação no banco temporário: %w", err)
	}
	triggersToRestore := append([]string(nil), meta.ActiveTriggers...)
	defer func() {
		_ = enableStageTriggers(context.Background(), c, stagePath, triggersToRestore)
	}()

	fullScript, err := copyRelationBlock(meta, c, stagePath, "", true)
	if err != nil {
		return result, err
	}
	appendLog("Tentando recuperar todos os registros legíveis de " + relation + " por leitura NATURAL...")
	out, fullErr := runISQLScript(ctx, c.FbSource, c.SourceVersion, sourcePort, c.User, c.Password, workingPath, "Resgatando registros de "+relation+" sem usar índices...", fullScript)
	if fullErr == nil {
		result.Recovered = parseRecoveredRows(out)
		result.Complete = true
	} else {
		appendLog("A leitura NATURAL completa falhou; tentando isolar registros pela chave primária.")
		if err := clearRelationOnStage(ctx, c, relation, stagePath); err != nil {
			return result, fmt.Errorf("falha ao limpar o banco temporário antes do isolamento: %w", err)
		}
		minValue, maxValue, boundsErr := primaryKeyBounds(ctx, c, sourcePort, workingPath, meta)
		if boundsErr != nil {
			return result, fmt.Errorf("não foi possível iniciar o isolamento por chave primária: %w", boundsErr)
		}
		attempts := 0
		rows, lost, rangeErr := rescueRanges(ctx, c, sourcePort, workingPath, stagePath, meta, minValue, maxValue, &attempts, 512)
		if rangeErr != nil {
			return result, rangeErr
		}
		result.Recovered = rows
		result.LostKeys = lost
		result.Complete = len(lost) == 0
	}

	if err := enableStageTriggers(ctx, c, stagePath, triggersToRestore); err != nil {
		return result, fmt.Errorf("os registros foram copiados, mas não foi possível reativar os triggers: %w", err)
	}
	triggersToRestore = nil
	if len(indexesToReactivate) > 0 {
		if _, err := runStageISQL(ctx, c, stagePath, "Reconstruindo índices no banco temporário após o resgate...", buildAlterIndexesScript(indexesToReactivate, true)); err != nil {
			return result, fmt.Errorf("os registros foram copiados, mas não foi possível reconstruir os índices no banco temporário: %w", err)
		}
	}

	newBackup := finalBackup
	if strings.TrimSpace(newBackup) == "" || strings.EqualFold(filepath.Clean(newBackup), filepath.Clean(baseBackup)) {
		newBackup = recoveredBackupPath(baseBackup)
	}
	if err := backupDatabasePath(ctx, c, stagePath, newBackup, "Gerando novo backup com os registros recuperados..."); err != nil {
		return result, fmt.Errorf("falha ao gerar backup pós-resgate: %w", err)
	}
	result.Backup = newBackup
	return result, nil
}

func recoveryStatsComparison(before, after recoveryStats) string {
	return fmt.Sprintf(
		"Antes: registros=%d | índices=%d | transações=%d | BLOBs=%d | páginas=%d\nDepois: registros=%d | índices=%d | transações=%d | BLOBs=%d | páginas=%d",
		before.RecordErrors, before.IndexErrors, before.TransactionErrors, before.BlobErrors, before.DatabaseErrors,
		after.RecordErrors, after.IndexErrors, after.TransactionErrors, after.BlobErrors, after.DatabaseErrors,
	)
}

func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func fileCopyWithSHA256(source, dest string) (string, int64, error) {
	in, err := os.Open(source)
	if err != nil {
		return "", 0, err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return "", 0, err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", 0, err
	}
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(dest)
		}
	}()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(out, h), in)
	if err != nil {
		return "", n, err
	}
	if err := out.Sync(); err != nil {
		return "", n, err
	}
	if err := out.Close(); err != nil {
		return "", n, err
	}
	ok = true
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

type recoveryReport struct {
	Started                   time.Time
	Finished                  time.Time
	Source                    string
	SourceVersion             string
	TargetVersion             string
	WorkingCopy               string
	Backup                    string
	MetadataBackup            string
	TestRestore               string
	Destination               string
	ReportPath                string
	TechnicalLog              string
	OriginalSHA256            string
	OriginalSize              int64
	Diagnostic                recoveryStats
	DiagnosticDone            bool
	PreMendDiagnostic         recoveryStats
	PostMendDiagnostic        recoveryStats
	PostMendChecked           bool
	LimboCount                int
	SkipData                  string
	SuspectedRelation         string
	SuspectedRelationSource   string
	EngineCrashRelation       string
	IndexRescueAttempted      bool
	IndexesDisabled           []string
	RowRescueAttempted        bool
	RowRescueRelation         string
	RowRescueRecovered        int64
	RowRescueLostKeys         []string
	RowRescueComplete         bool
	RowRescueStage            string
	BasePartialBackup         string
	DuplicateConflictDetected bool
	DuplicateFailedRestore    string
	DuplicateConflictSource   string
	DuplicateConflictWork     string
	DuplicateConflictReport   string
	DuplicateRepairedBackup   string
	DuplicateRepairApplied    bool
	DuplicateRowsRemoved      int64
	DuplicateConflicts        []restoreDuplicateConflict
	OrphanFKDetected          bool
	NoValidityRestore         bool
	ConstraintsScript         string
	ConstraintsDump           string
	ConstraintsDialect        string
	ConstraintsExtracted      int
	OrphanRowsScript          string
	CleanRebuildAttempted     bool
	CleanRebuildRelation      string
	CleanRebuildStage         string
	CleanRebuildBackup        string
	BackupFailureDetails      string
	FirebirdLogExcerpt        string
	TestRestoreState          string
	TestODS                   string
	TestDialect               string
	TestPageSize              string
	HealthReport              string
	Health                    postRecoveryHealth
	HealthChecked             bool
	FinalState                string
	PartialRecovery           bool
	Actions                   []string
	Warnings                  []string
	Errors                    []string
}

func (r *recoveryReport) action(text string) {
	r.Actions = append(r.Actions, text)
}

func (r *recoveryReport) warning(text string) {
	r.Warnings = append(r.Warnings, text)
}

func (r *recoveryReport) fail(text string) {
	r.Errors = append(r.Errors, text)
}

// noValidityConstraintWarnings monta os avisos do restore com -n: o banco
// restaurado nao tem as constraints e o operador precisa saber onde estao os
// scripts para inspecionar as linhas orfas e recriar as constraints depois.
func noValidityConstraintWarnings(r recoveryReport) []string {
	if !r.NoValidityRestore {
		return nil
	}
	warnings := []string{
		"O banco foi restaurado com -n (-no_validity): as FOREIGN KEY e as constraints CHECK NÃO existem no banco restaurado",
	}
	switch {
	case r.ConstraintsScript != "":
		warnings = append(warnings, fmt.Sprintf("DDL das constraints originais preservado em %s (%d constraint(s)); execute-o para recriá-las", r.ConstraintsScript, r.ConstraintsExtracted))
	case r.ConstraintsDump != "":
		warnings = append(warnings, "Não foi possível montar o DDL das constraints; o schema completo do banco de origem ficou em "+r.ConstraintsDump+", de onde os ALTER TABLE ... ADD CONSTRAINT devem ser extraídos manualmente")
	default:
		warnings = append(warnings, "Não foi possível extrair o DDL das constraints do banco de origem; recupere a definição delas de um backup anterior antes de recriá-las")
	}
	if r.OrphanRowsScript != "" {
		warnings = append(warnings, "Linhas órfãs: execute "+r.OrphanRowsScript+" para listar os filhos sem pai; inspecione os resultados antes de apagar ou corrigir qualquer linha")
	}
	warnings = append(warnings, "Recrie as constraints somente depois de sanear as linhas órfãs; enquanto isso não for feito o banco NÃO é apto para produção")
	return warnings
}

func writeRecoveryReport(r recoveryReport) error {
	if r.ReportPath == "" {
		return errors.New("caminho do relatório não definido")
	}
	var b strings.Builder
	b.WriteString("FIREBIRD TOOLBOX - RELATÓRIO DE RECUPERAÇÃO\r\n")
	b.WriteString("Versão do utilitário: " + appVersion + "\r\n")
	b.WriteString("Início: " + r.Started.Format("2006-01-02 15:04:05") + "\r\n")
	if !r.Finished.IsZero() {
		b.WriteString("Fim: " + r.Finished.Format("2006-01-02 15:04:05") + "\r\n")
		b.WriteString("Duração total: " + formatElapsedDuration(r.Finished.Sub(r.Started)) + "\r\n")
	}

	b.WriteString("\r\nORIGEM\r\n")
	b.WriteString("Arquivo: " + r.Source + "\r\n")
	b.WriteString("Firebird: " + r.SourceVersion + "\r\n")
	if r.OriginalSHA256 != "" {
		b.WriteString("SHA-256: " + r.OriginalSHA256 + "\r\n")
		b.WriteString(fmt.Sprintf("Tamanho: %d bytes\r\n", r.OriginalSize))
	}

	b.WriteString("\r\nDIAGNÓSTICO INICIAL\r\n")
	// Contador zerado não é banco saudável quando o diagnóstico nem chegou a
	// rodar. Dizer "nenhuma corrupção" nesse caso é afirmar o que não se sabe.
	if !r.DiagnosticDone {
		b.WriteString("Classificação: DIAGNÓSTICO NÃO CONCLUÍDO - o estado do banco não foi avaliado\r\n")
		b.WriteString("Os contadores abaixo não foram medidos e não devem ser lidos como ausência de corrupção.\r\n")
	} else {
		b.WriteString("Classificação: " + r.Diagnostic.Summary() + "\r\n")
		b.WriteString(fmt.Sprintf("Erros de registro: %d\r\n", r.Diagnostic.RecordErrors))
		b.WriteString(fmt.Sprintf("Erros de índice: %d\r\n", r.Diagnostic.IndexErrors))
		b.WriteString(fmt.Sprintf("Erros de transação: %d\r\n", r.Diagnostic.TransactionErrors))
		b.WriteString(fmt.Sprintf("Erros de BLOB: %d\r\n", r.Diagnostic.BlobErrors))
		b.WriteString(fmt.Sprintf("Erros de página: %d\r\n", r.Diagnostic.DatabaseErrors))
		b.WriteString(fmt.Sprintf("Transações em limbo detectadas: %d\r\n", r.LimboCount))
	}

	if r.PostMendChecked {
		b.WriteString("\r\nDIAGNÓSTICO PÓS-MEND\r\n")
		b.WriteString("Classificação após mend: " + r.PostMendDiagnostic.Summary() + "\r\n")
		comparison := recoveryStatsComparison(r.PreMendDiagnostic, r.PostMendDiagnostic)
		b.WriteString(strings.ReplaceAll(comparison, "\n", "\r\n") + "\r\n")
	}

	if r.SkipData != "" {
		b.WriteString("\r\nRECUPERAÇÃO PARCIAL / DADOS EXCLUÍDOS\r\n")
		b.WriteString("Regex usada no -skip_data: " + r.SkipData + "\r\n")
		if r.SuspectedRelation != "" {
			b.WriteString("Relação suspeita: " + r.SuspectedRelation + "\r\n")
		}
		b.WriteString("ATENÇÃO: os dados das relações correspondentes não fazem parte do backup recuperado.\r\n")
	}

	if r.IndexRescueAttempted {
		b.WriteString("\r\nRESGATE POR ÍNDICES\r\n")
		if len(r.IndexesDisabled) > 0 {
			b.WriteString("Índices desativados somente na cópia de trabalho: " + strings.Join(r.IndexesDisabled, ", ") + "\r\n")
		} else {
			b.WriteString("Nenhum índice pôde ser desativado com segurança na cópia de trabalho.\r\n")
		}
	}

	if r.RowRescueAttempted {
		b.WriteString("\r\nRESGATE DE REGISTROS DA RELAÇÃO\r\n")
		b.WriteString("Relação: " + r.RowRescueRelation + "\r\n")
		b.WriteString(fmt.Sprintf("Registros recuperados: %d\r\n", r.RowRescueRecovered))
		if r.RowRescueComplete {
			b.WriteString("Estado do resgate da relação: COMPLETO\r\n")
		} else {
			b.WriteString("Estado do resgate da relação: PARCIAL\r\n")
		}
		if len(r.RowRescueLostKeys) > 0 {
			b.WriteString("Chaves primárias não recuperadas: " + strings.Join(r.RowRescueLostKeys, ", ") + "\r\n")
		}
		if r.RowRescueStage != "" {
			b.WriteString("Banco temporário usado para o resgate: " + r.RowRescueStage + "\r\n")
		}
	}

	if r.DuplicateConflictDetected {
		b.WriteString("\r\nCONFLITOS DE CHAVE PRIMÁRIA / ÚNICA\r\n")
		for _, conflict := range r.DuplicateConflicts {
			b.WriteString("Tabela: " + conflict.Relation + "\r\n")
			b.WriteString("Constraint: " + conflict.Constraint + "\r\n")
			if conflict.IndexName != "" {
				b.WriteString("Índice: " + conflict.IndexName + "\r\n")
			}
			b.WriteString("Chave problemática: " + duplicateConflictKeyLabel(conflict) + "\r\n")
			b.WriteString(fmt.Sprintf("Ocorrências: %d | Extras removidas na cópia: %d\r\n", conflict.Occurrences, conflict.RowsRemoved))
			if conflict.Identical {
				b.WriteString("Comparação: duplicidade estrutural idêntica nas colunas comparáveis; uma ocorrência foi preservada.\r\n")
			} else if len(conflict.DifferingColumns) > 0 {
				b.WriteString("Campos diferentes: " + strings.Join(conflict.DifferingColumns, ", ") + "\r\n")
			} else if conflict.ComparisonInconclusive {
				b.WriteString("Comparação: inconclusiva; decisão explícita necessária antes de remover duplicidades.\r\n")
			}
			if len(conflict.BlobComparison) > 0 {
				b.WriteString("Comparação de BLOBs/Fotos: " + strings.Join(conflict.BlobComparison, "; ") + "\r\n")
			}
			if conflict.ComparisonFile != "" {
				b.WriteString("Comparação exportada: " + conflict.ComparisonFile + "\r\n")
			}
			if conflict.ChosenOccurrence > 0 {
				b.WriteString(fmt.Sprintf("Decisão: usuário escolheu manter o REGISTRO %d somente na cópia de trabalho.\r\n", conflict.ChosenOccurrence))
			} else if conflict.UserAcceptedDataChoice {
				b.WriteString("Decisão: usuário autorizou saneamento explícito somente na cópia de trabalho.\r\n")
			}
			b.WriteString("\r\n")
		}
		if r.DuplicateRowsRemoved > 0 {
			b.WriteString(fmt.Sprintf("Total de ocorrências extras removidas na cópia de trabalho: %d\r\n", r.DuplicateRowsRemoved))
		}
	}

	if r.BackupFailureDetails != "" || r.SuspectedRelation != "" || r.EngineCrashRelation != "" {
		b.WriteString("\r\nFALHA DO BACKUP / DIAGNÓSTICO DO GBAK\r\n")
		if r.SuspectedRelation != "" {
			line := "Possível relação afetada: " + r.SuspectedRelation
			if r.SuspectedRelationSource != "" {
				line += " (identificada pelo " + r.SuspectedRelationSource + ")"
			}
			b.WriteString(line + "\r\n")
		}
		if r.EngineCrashRelation != "" {
			b.WriteString("Tabela onde o motor sofreu bugcheck: " + r.EngineCrashRelation + " (firebird.log)\r\n")
			if r.SuspectedRelation != "" && !strings.EqualFold(r.EngineCrashRelation, r.SuspectedRelation) {
				b.WriteString("As duas tabelas são diferentes: o -skip_data trabalha sobre a relação que bloqueia o backup, enquanto o bugcheck indica outro ponto de dano. Não trate nenhuma das duas como a única causa.\r\n")
			}
		}
		if r.BackupFailureDetails != "" {
			b.WriteString(strings.ReplaceAll(r.BackupFailureDetails, "\n", "\r\n") + "\r\n")
		}
	}

	b.WriteString("\r\nARQUIVOS DE RECUPERAÇÃO\r\n")
	if r.WorkingCopy != "" {
		b.WriteString("Cópia de trabalho: " + r.WorkingCopy + "\r\n")
	}
	if r.Backup != "" {
		b.WriteString("Backup: " + r.Backup + "\r\n")
	}
	if r.MetadataBackup != "" {
		b.WriteString("Backup somente de metadados: " + r.MetadataBackup + "\r\n")
	}
	if r.BasePartialBackup != "" {
		b.WriteString("Backup-base sem a relação problemática: " + r.BasePartialBackup + "\r\n")
	}
	if r.TestRestore != "" {
		b.WriteString("Restore de teste: " + r.TestRestore + "\r\n")
	}
	if r.Destination != "" {
		b.WriteString("Destino final: " + r.Destination + "\r\n")
	}
	if r.TechnicalLog != "" {
		b.WriteString("Log técnico completo: " + r.TechnicalLog + "\r\n")
	}
	if r.DuplicateFailedRestore != "" {
		b.WriteString("Restore originalmente falho preservado: " + r.DuplicateFailedRestore + "\r\n")
	}
	if r.DuplicateConflictSource != "" {
		b.WriteString("Banco de inspeção com índices inativos preservado: " + r.DuplicateConflictSource + "\r\n")
	}
	if r.DuplicateConflictWork != "" {
		b.WriteString("Banco saneado de trabalho: " + r.DuplicateConflictWork + "\r\n")
	}
	if r.DuplicateConflictReport != "" {
		b.WriteString("Relatório de conflitos de chave: " + r.DuplicateConflictReport + "\r\n")
	}
	if r.DuplicateRepairedBackup != "" {
		b.WriteString("Backup após saneamento de duplicidades: " + r.DuplicateRepairedBackup + "\r\n")
	}
	if r.CleanRebuildStage != "" {
		b.WriteString("Banco de reconstrução limpa: " + r.CleanRebuildStage + "\r\n")
	}
	if r.CleanRebuildBackup != "" {
		b.WriteString("Backup após reconstrução limpa de " + r.CleanRebuildRelation + ": " + r.CleanRebuildBackup + "\r\n")
	}
	if r.ConstraintsScript != "" {
		b.WriteString("Script com as constraints originais: " + r.ConstraintsScript + "\r\n")
	}
	if r.ConstraintsDump != "" {
		b.WriteString("Schema completo extraído com isql -x: " + r.ConstraintsDump + "\r\n")
	}
	if r.OrphanRowsScript != "" {
		b.WriteString("Script de inspeção de linhas órfãs: " + r.OrphanRowsScript + "\r\n")
	}
	if r.HealthReport != "" {
		b.WriteString("Relatório de saúde e pós-recuperação: " + r.HealthReport + "\r\n")
	}

	if r.HealthChecked {
		b.WriteString("\r\nPÓS-RECUPERAÇÃO / SAÚDE\r\n")
		b.WriteString("Estado de saúde: " + r.Health.Readiness + "\r\n")
		b.WriteString(fmt.Sprintf("Banco acessível: %t | Sweep final: %t | Validação estrutural: %t\r\n", r.Health.ConnectivityOK, r.Health.SweepOK, r.Health.ValidationOK))
		b.WriteString(fmt.Sprintf("Tabelas: %d | Índices: %d | PRIMARY KEY/UNIQUE: %d | Índices inativos: %d | Constraints inativas: %d\r\n", r.Health.UserTableCount, r.Health.UserIndexCount, r.Health.PrimaryUniqueCount, len(r.Health.InactiveIndexes), len(r.Health.InactiveConstraints)))
		if len(r.Health.CriticalIssues) > 0 {
			b.WriteString("Gates críticos reprovados: " + strings.Join(r.Health.CriticalIssues, " | ") + "\r\n")
		}
	}

	b.WriteString("\r\nAÇÕES EXECUTADAS\r\n")
	if len(r.Actions) == 0 {
		b.WriteString("- Nenhuma ação registrada.\r\n")
	}
	for _, item := range r.Actions {
		b.WriteString("- " + item + "\r\n")
	}
	warnings := append([]string(nil), r.Warnings...)
	warnings = append(warnings, noValidityConstraintWarnings(r)...)
	if len(warnings) > 0 {
		b.WriteString("\r\nAVISOS\r\n")
		for _, item := range warnings {
			b.WriteString("- " + item + "\r\n")
		}
	}
	if len(r.Errors) > 0 {
		b.WriteString("\r\nERROS\r\n")
		for _, item := range r.Errors {
			b.WriteString("- " + item + "\r\n")
		}
	}

	if r.FirebirdLogExcerpt != "" {
		b.WriteString("\r\nFIREBIRD.LOG - TRECHO GERADO DURANTE A RECUPERAÇÃO\r\n")
		b.WriteString(strings.ReplaceAll(r.FirebirdLogExcerpt, "\n", "\r\n") + "\r\n")
	}

	b.WriteString("\r\nRESULTADO\r\n")
	if r.TestRestoreState != "" {
		b.WriteString("Restore de teste: " + r.TestRestoreState + "\r\n")
	}
	if r.TestODS != "" {
		b.WriteString("ODS validado: " + r.TestODS + "\r\n")
	}
	if r.TestDialect != "" {
		b.WriteString("Dialeto validado: " + r.TestDialect + "\r\n")
	}
	if r.TestPageSize != "" {
		b.WriteString("Tamanho de página validado: " + r.TestPageSize + " bytes\r\n")
	}
	if r.FinalState != "" {
		b.WriteString("Estado final: " + r.FinalState + "\r\n")
	}
	return os.WriteFile(r.ReportPath, []byte(b.String()), 0600)
}

func recoveryWorkspace(c workflowConfig) (dir, copyPath, backupPath, testPath, reportPath string, err error) {
	base := strings.TrimSuffix(filepath.Base(c.Source), filepath.Ext(c.Source))
	if base == "" {
		base = "DBCOM"
	}
	stamp := time.Now().Format("20060102_150405_000")
	dir = filepath.Join(c.BackupDir, "RECOVERY_"+base+"_"+stamp)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return
	}
	ext := filepath.Ext(c.Source)
	if ext == "" {
		ext = ".FDB"
	}
	copyPath = filepath.Join(dir, base+"_WORK"+ext)
	backupPath = filepath.Join(dir, base+"_RECOVERY.BAK")
	testPath = filepath.Join(dir, base+"_RESTORE_TEST"+ext)
	reportPath = filepath.Join(dir, "RELATORIO_RECUPERACAO.txt")
	return
}

func countLimboTransactions(output string) int {
	re := regexp.MustCompile(`(?im)^\s*Transaction\s+\d+\s+is in limbo\.`)
	return len(re.FindAllString(output, -1))
}

func gfixMendArgs(user, password, dbConn string) []string {
	// O nome completo -mend é aceito nas linhas atuais; nas antigas corresponde a -m[end].
	return []string{"-user", user, "-pass", password, "-mend", "-ignore", dbConn}
}

func normalizeSkipData(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = strings.TrimSpace(value[1 : len(value)-1])
		}
	}
	return value
}

func advancedBackupArgs(c workflowConfig, dbConn, backup string, useIgnore, ignoreLimbo bool) []string {
	args := []string{"-b", "-v"}
	if useIgnore {
		args = append(args, "-ignore")
	}
	if ignoreLimbo {
		args = append(args, "-limbo")
	}
	if skip := normalizeSkipData(c.SkipData); skip != "" && versionNumber(c.SourceVersion) >= 3.0 {
		args = append(args, "-skip_data", skip)
	}
	args = append(args, "-user", c.User, "-pass", c.Password, dbConn, backup)
	return args
}

func testRestoreEngine(c workflowConfig) (version, root, port string) {
	if c.TargetVersion != "" {
		return c.TargetVersion, c.FbTarget, c.TargetPort
	}
	return c.SourceVersion, c.FbSource, c.SourcePort
}

// restoreTestArgs monta a linha do gbak de restauracao. noValidity acrescenta
// -n (-no_validity), que restaura sem recriar as constraints de validade e as
// FOREIGN KEY. E destrutivo nos metadados do banco restaurado: as constraints
// precisam ser recriadas a mao depois de sanear as linhas orfas.
func restoreTestArgs(version, port, user, password, backup, dest string, oneAtATime, noValidity bool) []string {
	flags := []string{"-c", "-v"}
	if oneAtATime {
		flags = append(flags, "-one_at_a_time")
	}
	if noValidity {
		flags = append(flags, "-n")
	}
	if supportsServiceManagerRestore(version) {
		return append(flags, "-se", serviceManager(port), "-user", user, "-password", password, backup, dest)
	}
	return append(flags, "-user", user, "-password", password, backup, dest)
}

func restoreInactiveArgs(version, port, user, password, backup, dest string) []string {
	flags := []string{"-c", "-v", "-inactive"}
	if supportsServiceManagerRestore(version) {
		return append(flags, "-se", serviceManager(port), "-user", user, "-password", password, backup, dest)
	}
	return append(flags, "-user", user, "-password", password, backup, dest)
}

func restoreBackupWithInactiveIndexes(ctx context.Context, root, version, port, user, password, backup, dest string) error {
	gbak := firebirdExe(root, version, "gbak.exe")
	if !fileExists(gbak) {
		return fmt.Errorf("não encontrei gbak.exe do Firebird %s", version)
	}
	_ = os.Remove(dest)
	_, err := runRecoveryCapture(ctx, "Restaurando cópia de inspeção com todos os índices inativos...", gbak, restoreInactiveArgs(version, port, user, password, backup, dest)...)
	if err != nil {
		return err
	}
	if !fileExists(dest) {
		return fmt.Errorf("o restore com índices inativos terminou sem criar %s", dest)
	}
	return nil
}

func inactiveUserIndexes(ctx context.Context, root, version, port, user, password, dbPath string) ([]string, error) {
	script := "SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\nSELECT 'FTB_INACTIVE|' || TRIM(RDB$INDEX_NAME) FROM RDB$INDICES WHERE COALESCE(RDB$SYSTEM_FLAG,0) = 0 AND COALESCE(RDB$INDEX_INACTIVE,0) <> 0 ORDER BY RDB$INDEX_NAME;\nQUIT;\n"
	out, err := runISQLScript(ctx, root, version, port, user, password, dbPath, "Listando índices inativos no banco de saneamento...", script)
	if err != nil {
		return nil, err
	}
	return markerLines(out, "FTB_INACTIVE|"), nil
}

func activateAllInactiveUserIndexes(ctx context.Context, root, version, port, user, password, dbPath string) error {
	remaining, err := inactiveUserIndexes(ctx, root, version, port, user, password, dbPath)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return nil
	}

	// Constraints restored with -inactive can depend on other constraint indexes.
	// Repeating activation passes avoids treating a temporary dependency ordering
	// failure (for example, a foreign key attempted before its referenced unique
	// index) as permanent corruption.
	previousCount := len(remaining) + 1
	for pass := 1; len(remaining) > 0 && pass <= 6; pass++ {
		if len(remaining) >= previousCount {
			break
		}
		previousCount = len(remaining)
		_, _ = runISQLScript(ctx, root, version, port, user, password, dbPath,
			fmt.Sprintf("Reconstruindo e reativando índices do banco saneado (passo %d)...", pass),
			buildAlterIndexesScript(remaining, true))
		remaining, err = inactiveUserIndexes(ctx, root, version, port, user, password, dbPath)
		if err != nil {
			return err
		}
	}
	if len(remaining) > 0 {
		shown := remaining
		if len(shown) > 12 {
			shown = shown[:12]
		}
		return fmt.Errorf("%d índice(s) continuaram inativos após o saneamento: %s", len(remaining), strings.Join(shown, ", "))
	}
	return nil
}

type inactiveIndexDetail struct {
	IndexName      string
	Relation       string
	ConstraintName string
	ConstraintType string
}

func inactiveIndexDetails(ctx context.Context, root, version, port, user, password, dbPath string) ([]inactiveIndexDetail, error) {
	script := "SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\n" +
		"SELECT 'FTB_IIDX|' || TRIM(i.RDB$INDEX_NAME) || '|' || TRIM(i.RDB$RELATION_NAME) || '|' || COALESCE(TRIM(rc.RDB$CONSTRAINT_NAME), '') || '|' || COALESCE(TRIM(rc.RDB$CONSTRAINT_TYPE), '') " +
		"FROM RDB$INDICES i LEFT JOIN RDB$RELATION_CONSTRAINTS rc ON rc.RDB$INDEX_NAME = i.RDB$INDEX_NAME " +
		"WHERE COALESCE(i.RDB$SYSTEM_FLAG,0) = 0 AND COALESCE(i.RDB$INDEX_INACTIVE,0) <> 0 ORDER BY i.RDB$RELATION_NAME, i.RDB$INDEX_NAME;\nQUIT;\n"
	out, err := runISQLScript(ctx, root, version, port, user, password, dbPath, "Mapeando índices inativos restantes...", script)
	if err != nil {
		return nil, err
	}
	var result []inactiveIndexDetail
	for _, line := range markerLines(out, "FTB_IIDX|") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 2 {
			continue
		}
		d := inactiveIndexDetail{IndexName: strings.TrimSpace(parts[0]), Relation: strings.TrimSpace(parts[1])}
		if len(parts) >= 3 {
			d.ConstraintName = strings.TrimSpace(parts[2])
		}
		if len(parts) >= 4 {
			d.ConstraintType = strings.TrimSpace(parts[3])
		}
		if d.IndexName != "" && d.Relation != "" {
			result = append(result, d)
		}
	}
	return result, nil
}

func activateInactiveIndexesDetectingConflicts(ctx context.Context, root, version, port, user, password, dbPath string) ([]inactiveIndexDetail, []restoreDuplicateConflict, error) {
	details, err := inactiveIndexDetails(ctx, root, version, port, user, password, dbPath)
	if err != nil {
		return nil, nil, err
	}
	if len(details) == 0 {
		return nil, nil, nil
	}
	var lastFailure string
	for pass := 1; pass <= 12; pass++ {
		progress := false
		for _, detail := range details {
			script := "SET BAIL ON;\nALTER INDEX " + sqlIdentifier(detail.IndexName) + " ACTIVE;\nCOMMIT;\nQUIT;\n"
			out, activateErr := runISQLScript(ctx, root, version, port, user, password, dbPath, fmt.Sprintf("Tentando reconstruir índice %s (passo %d)...", detail.IndexName, pass), script)
			if activateErr == nil {
				progress = true
				continue
			}
			if conflicts := parseRestoreDuplicateConflicts(out); len(conflicts) > 0 {
				for i := range conflicts {
					if conflicts[i].IndexName == "" {
						conflicts[i].IndexName = detail.IndexName
					}
				}
				return details, conflicts, nil
			}
			if strings.TrimSpace(out) != "" {
				lastFailure = tailText(out, 16)
			} else {
				lastFailure = activateErr.Error()
			}
		}
		remaining, mapErr := inactiveIndexDetails(ctx, root, version, port, user, password, dbPath)
		if mapErr != nil {
			return nil, nil, mapErr
		}
		if len(remaining) == 0 {
			return nil, nil, nil
		}
		if !progress && len(remaining) >= len(details) {
			details = remaining
			break
		}
		details = remaining
	}
	if lastFailure != "" {
		writeRecoveryTechLine("ÍNDICES AINDA INATIVOS APÓS TENTATIVAS: " + lastFailure)
	}
	return details, nil, nil
}

func uniqueRelationsFromInactive(details []inactiveIndexDetail) []string {
	seen := map[string]bool{}
	var result []string
	for _, detail := range details {
		relation := strings.TrimSpace(detail.Relation)
		if relation == "" || strings.HasPrefix(strings.ToUpper(relation), "RDB$") || strings.HasPrefix(strings.ToUpper(relation), "MON$") || strings.HasPrefix(strings.ToUpper(relation), "SEC$") {
			continue
		}
		key := strings.ToUpper(relation)
		if !seen[key] {
			seen[key] = true
			result = append(result, relation)
		}
	}
	return result
}

type duplicateKeyPart struct {
	Column   string
	RawValue string
	IsNull   bool
}

type restoreDuplicateConflict struct {
	Constraint             string
	Relation               string
	KeyParts               []duplicateKeyPart
	IndexName              string
	Occurrences            int64
	DifferingColumns       []string
	InconclusiveColumns    []string
	ComparisonInconclusive bool
	Identical              bool
	UserAcceptedDataChoice bool
	ChosenOccurrence       int
	ComparisonFile         string
	BlobComparison         []string
	RowsRemoved            int64
}

func splitKeyAssignments(text string) []string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "(")
	text = strings.TrimSuffix(text, ")")
	var result []string
	start := 0
	inSingle, inDouble := false, false
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\'':
			if !inDouble {
				if inSingle && i+1 < len(text) && text[i+1] == '\'' {
					i++
					continue
				}
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case ',':
			if !inSingle && !inDouble {
				part := strings.TrimSpace(text[start:i])
				if part != "" {
					result = append(result, part)
				}
				start = i + 1
			}
		}
	}
	if tail := strings.TrimSpace(text[start:]); tail != "" {
		result = append(result, tail)
	}
	return result
}

func parseProblematicKey(text string) []duplicateKeyPart {
	var result []duplicateKeyPart
	assignRe := regexp.MustCompile(`^\s*"([^"]+)"\s*=\s*(.+?)\s*$`)
	for _, item := range splitKeyAssignments(text) {
		m := assignRe.FindStringSubmatch(item)
		if len(m) != 3 {
			continue
		}
		raw := strings.TrimSpace(m[2])
		result = append(result, duplicateKeyPart{
			Column:   strings.TrimSpace(m[1]),
			RawValue: raw,
			IsNull:   strings.EqualFold(raw, "NULL") || strings.EqualFold(raw, "<NULL>"),
		})
	}
	return result
}

func duplicateConflictKeyLabel(conflict restoreDuplicateConflict) string {
	var parts []string
	for _, key := range conflict.KeyParts {
		parts = append(parts, key.Column+"="+key.RawValue)
	}
	return strings.Join(parts, " / ")
}

func parseRestoreDuplicateConflicts(output string) []restoreDuplicateConflict {
	constraintRe := regexp.MustCompile(`(?i)violation of PRIMARY or UNIQUE KEY constraint\s+"([^"]+)"\s+on table\s+"([^"]+)"`)
	keyRe := regexp.MustCompile(`(?i)Problematic key value is\s*(\(.+\))`)
	var pending *restoreDuplicateConflict
	var result []restoreDuplicateConflict
	seen := map[string]bool{}
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if m := constraintRe.FindStringSubmatch(line); len(m) == 3 {
			pending = &restoreDuplicateConflict{Constraint: strings.TrimSpace(m[1]), Relation: strings.TrimSpace(m[2])}
			continue
		}
		if pending == nil {
			continue
		}
		if m := keyRe.FindStringSubmatch(line); len(m) == 2 {
			pending.KeyParts = parseProblematicKey(m[1])
			if len(pending.KeyParts) == 0 {
				pending = nil
				continue
			}
			key := strings.ToUpper(pending.Relation + "|" + pending.Constraint + "|" + duplicateConflictKeyLabel(*pending))
			if !seen[key] {
				seen[key] = true
				result = append(result, *pending)
			}
			pending = nil
		}
	}
	return result
}

func duplicateKeySQLValue(part duplicateKeyPart) (string, error) {
	if part.IsNull {
		return "NULL", nil
	}
	raw := strings.TrimSpace(part.RawValue)
	if raw == "" || strings.ContainsAny(raw, "\r\n;") || strings.Contains(raw, "--") || strings.Contains(raw, "/*") {
		return "", fmt.Errorf("valor de chave inseguro ou vazio para %s", part.Column)
	}
	if regexp.MustCompile(`^[+-]?(?:\d+\.?\d*|\d*\.\d+)(?:[eE][+-]?\d+)?$`).MatchString(raw) {
		return raw, nil
	}
	if regexp.MustCompile(`(?i)^x'[0-9a-f]+'$`).MatchString(raw) {
		return raw, nil
	}
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return raw, nil
	}
	trimmed := strings.Trim(raw, `"`)
	return sqlStringLiteral(trimmed), nil
}

func duplicateKeyPredicate(conflict restoreDuplicateConflict, alias string) (string, error) {
	if len(conflict.KeyParts) == 0 {
		return "", errors.New("conflito sem valores de chave")
	}
	prefix := ""
	if strings.TrimSpace(alias) != "" {
		prefix = alias + "."
	}
	var parts []string
	for _, key := range conflict.KeyParts {
		column := prefix + sqlIdentifier(key.Column)
		if key.IsNull {
			parts = append(parts, column+" IS NULL")
			continue
		}
		value, err := duplicateKeySQLValue(key)
		if err != nil {
			return "", err
		}
		parts = append(parts, column+" = "+value)
	}
	return strings.Join(parts, " AND "), nil
}

func parseMarkerInt64(output, marker string) int64 {
	for _, line := range markerLines(output, marker) {
		if n, err := strconv.ParseInt(strings.TrimSpace(line), 10, 64); err == nil {
			return n
		}
	}
	return 0
}

func duplicateOccurrenceCount(ctx context.Context, root, version, port, user, password, dbPath string, conflict restoreDuplicateConflict) (int64, error) {
	where, err := duplicateKeyPredicate(conflict, "T")
	if err != nil {
		return 0, err
	}
	script := "SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\nSELECT 'FTB_DUPCOUNT=' || CAST(COUNT(*) AS VARCHAR(30)) FROM " + sqlIdentifier(conflict.Relation) + " T WHERE " + where + ";\nQUIT;\n"
	out, err := runISQLScript(ctx, root, version, port, user, password, dbPath, "Contando ocorrências da chave duplicada em "+conflict.Relation+"...", script)
	if err != nil {
		return 0, err
	}
	return parseMarkerInt64(out, "FTB_DUPCOUNT="), nil
}

func columnIsConflictComparable(col relationColumn) bool {
	return !col.HasArray && col.FieldType != 261
}

func duplicateDifferenceScript(meta relationMetadata, conflict restoreDuplicateConflict) (string, []string, error) {
	whereA, err := duplicateKeyPredicate(conflict, "A")
	if err != nil {
		return "", nil, err
	}
	whereB, err := duplicateKeyPredicate(conflict, "B")
	if err != nil {
		return "", nil, err
	}
	var b strings.Builder
	b.WriteString("SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\n")
	var inconclusive []string
	for _, col := range meta.Columns {
		if !columnIsConflictComparable(col) {
			inconclusive = append(inconclusive, col.Name)
			continue
		}
		qcol := sqlIdentifier(col.Name)
		b.WriteString("SELECT 'FTB_DIFF|" + strings.ReplaceAll(col.Name, "'", "''") + "' FROM RDB$DATABASE WHERE EXISTS (SELECT FIRST 1 1 FROM " + sqlIdentifier(meta.Name) + " A, " + sqlIdentifier(meta.Name) + " B WHERE " + whereA + " AND " + whereB + " AND A.RDB$DB_KEY <> B.RDB$DB_KEY AND A." + qcol + " IS DISTINCT FROM B." + qcol + ");\n")
	}
	b.WriteString("QUIT;\n")
	return b.String(), inconclusive, nil
}

func analyzeDuplicateConflict(ctx context.Context, root, version, port, user, password, dbPath string, conflict restoreDuplicateConflict) (restoreDuplicateConflict, string, error) {
	meta, err := loadRelationMetadataAt(ctx, root, version, port, user, password, dbPath, conflict.Relation)
	if err != nil {
		return conflict, "", err
	}
	count, err := duplicateOccurrenceCount(ctx, root, version, port, user, password, dbPath, conflict)
	if err != nil {
		return conflict, "", err
	}
	conflict.Occurrences = count
	script, inconclusive, err := duplicateDifferenceScript(meta, conflict)
	if err != nil {
		return conflict, "", err
	}
	out, diffErr := runISQLScript(ctx, root, version, port, user, password, dbPath, "Comparando ocorrências da chave duplicada em "+conflict.Relation+"...", script)
	if diffErr != nil {
		conflict.ComparisonInconclusive = true
		conflict.InconclusiveColumns = append(conflict.InconclusiveColumns, "comparação SQL falhou: "+diffErr.Error())
	} else {
		conflict.DifferingColumns = markerLines(out, "FTB_DIFF|")
		conflict.InconclusiveColumns = inconclusive
		conflict.ComparisonInconclusive = len(inconclusive) > 0 && len(conflict.DifferingColumns) == 0
		conflict.Identical = len(conflict.DifferingColumns) == 0 && len(inconclusive) == 0
	}
	where, predErr := duplicateKeyPredicate(conflict, "T")
	if predErr != nil {
		return conflict, "", predErr
	}
	exportScript := "SET LIST ON;\nSET HEADING OFF;\nSET COUNT ON;\nSELECT * FROM " + sqlIdentifier(conflict.Relation) + " T WHERE " + where + ";\nQUIT;\n"
	export, exportErr := runISQLScript(ctx, root, version, port, user, password, dbPath, "Exportando ocorrências conflitantes de "+conflict.Relation+"...", exportScript)
	if exportErr != nil {
		export += "\n[Não foi possível exportar todas as colunas: " + exportErr.Error() + "]"
	}
	return conflict, export, nil
}

type duplicateOccurrenceDetails struct {
	Ordinal int
	Values  map[string]string
	Blobs   map[string]string
}

type conflictResolutionDecision int

const (
	conflictResolutionCancel conflictResolutionDecision = iota
	conflictResolutionKeep1
	conflictResolutionKeep2
)

type duplicateResolutionView struct {
	SummaryText string
	Record1Text string
	Record2Text string
	DetailsText string
	FilePath    string
}

func containsStringFold(items []string, value string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func removeStringFold(items []string, value string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(value)) {
			result = append(result, item)
		}
	}
	return result
}

func duplicateSummaryColumns(meta relationMetadata, conflict restoreDuplicateConflict) []relationColumn {
	wanted := []string{}
	for _, key := range conflict.KeyParts {
		wanted = append(wanted, key.Column)
	}
	wanted = append(wanted, conflict.DifferingColumns...)
	wanted = append(wanted,
		"COD_ESTQ", "NOM_DOCIT", "NOME", "DESCRICAO", "DESC_DOCIT", "QTDE_DOCIT", "QUANTIDADE",
		"PRECO_DOCIT", "PRECO", "TOTAL_DOCIT", "TOTAL", "UN_DOCIT", "NUMPED_DOC", "NUM_DOC",
		"EMISS_DOC", "DTINS_DOCIT", "DATA", "CODBAR_PROD",
	)
	var result []relationColumn
	seen := map[string]bool{}
	appendCol := func(col relationColumn) {
		key := strings.ToUpper(strings.TrimSpace(col.Name))
		if key == "" || seen[key] || col.HasArray || col.FieldType == 261 {
			return
		}
		seen[key] = true
		result = append(result, col)
	}
	for _, name := range wanted {
		for _, col := range meta.Columns {
			if strings.EqualFold(col.Name, name) {
				appendCol(col)
				break
			}
		}
	}
	for _, col := range meta.Columns {
		if len(result) >= 16 {
			break
		}
		appendCol(col)
	}
	return result
}

func duplicateOccurrenceDetailScript(meta relationMetadata, conflict restoreDuplicateConflict, ordinal int, columns []relationColumn, version string) (string, error) {
	where, err := duplicateKeyPredicate(conflict, "T")
	if err != nil {
		return "", err
	}
	skip := ordinal - 1
	if skip < 0 {
		skip = 0
	}
	var b strings.Builder
	b.WriteString("SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\n")
	for _, col := range columns {
		q := sqlIdentifier(col.Name)
		expr := "COALESCE(CAST(T." + q + " AS VARCHAR(1000)), '<null>')"
		b.WriteString(fmt.Sprintf("SELECT FIRST 1 SKIP %d 'FTB_VALUE|%s|' || %s FROM %s T WHERE %s PLAN (T NATURAL);\n", skip, strings.ReplaceAll(col.Name, "'", "''"), expr, sqlIdentifier(meta.Name), where))
	}
	if versionNumber(version) >= 4.0 {
		for _, col := range meta.Columns {
			if col.FieldType != 261 || col.HasArray {
				continue
			}
			q := sqlIdentifier(col.Name)
			expr := "CASE WHEN T." + q + " IS NULL THEN '<null>' ELSE CAST(HEX_ENCODE(CRYPT_HASH(T." + q + " USING SHA256)) AS VARCHAR(80)) END"
			b.WriteString(fmt.Sprintf("SELECT FIRST 1 SKIP %d 'FTB_BLOB|%s|' || %s FROM %s T WHERE %s PLAN (T NATURAL);\n", skip, strings.ReplaceAll(col.Name, "'", "''"), expr, sqlIdentifier(meta.Name), where))
		}
	}
	b.WriteString("QUIT;\n")
	return b.String(), nil
}

func loadDuplicateOccurrenceDetails(ctx context.Context, root, version, port, user, password, dbPath string, meta relationMetadata, conflict restoreDuplicateConflict, ordinal int) (duplicateOccurrenceDetails, error) {
	result := duplicateOccurrenceDetails{Ordinal: ordinal, Values: map[string]string{}, Blobs: map[string]string{}}
	script, err := duplicateOccurrenceDetailScript(meta, conflict, ordinal, duplicateSummaryColumns(meta, conflict), version)
	if err != nil {
		return result, err
	}
	out, err := runISQLScript(ctx, root, version, port, user, password, dbPath, fmt.Sprintf("Lendo detalhes da ocorrência %d de %s...", ordinal, conflict.Relation), script)
	if err != nil {
		return result, err
	}
	for _, line := range markerLines(out, "FTB_VALUE|") {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			result.Values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	for _, line := range markerLines(out, "FTB_BLOB|") {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			result.Blobs[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result, nil
}

func applyBlobComparison(conflict *restoreDuplicateConflict, first, second duplicateOccurrenceDetails, meta relationMetadata, version string) {
	if conflict == nil {
		return
	}
	var notes []string
	for _, col := range meta.Columns {
		if col.FieldType != 261 || col.HasArray {
			continue
		}
		name := col.Name
		h1, ok1 := first.Blobs[name]
		h2, ok2 := second.Blobs[name]
		if versionNumber(version) < 4.0 || !ok1 || !ok2 {
			notes = append(notes, name+": SHA-256 indisponível nesta versão do Firebird")
			continue
		}
		if h1 == h2 {
			notes = append(notes, name+": SHA-256 igual ("+h1+")")
			conflict.InconclusiveColumns = removeStringFold(conflict.InconclusiveColumns, name)
		} else {
			notes = append(notes, name+": SHA-256 DIFERENTE | registro 1="+h1+" | registro 2="+h2)
			if !containsStringFold(conflict.DifferingColumns, name) {
				conflict.DifferingColumns = append(conflict.DifferingColumns, name)
			}
			conflict.InconclusiveColumns = removeStringFold(conflict.InconclusiveColumns, name)
		}
	}
	conflict.BlobComparison = notes
	conflict.ComparisonInconclusive = len(conflict.InconclusiveColumns) > 0
	conflict.Identical = len(conflict.DifferingColumns) == 0 && len(conflict.InconclusiveColumns) == 0
}

func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if strings.ToUpper(keys[j]) < strings.ToUpper(keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func formatDuplicateOccurrence(details duplicateOccurrenceDetails, columns []relationColumn, conflict restoreDuplicateConflict) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("REGISTRO %d\r\n", details.Ordinal))
	b.WriteString(strings.Repeat("-", 36) + "\r\n")
	for _, col := range columns {
		if value, ok := details.Values[col.Name]; ok {
			prefix := ""
			if containsStringFold(conflict.DifferingColumns, col.Name) {
				prefix = "[DIFERENTE] "
			} else if containsStringFold(conflict.InconclusiveColumns, col.Name) {
				prefix = "[INCONCLUSIVO] "
			}
			b.WriteString(prefix + col.Name + ": " + value + "\r\n")
		}
	}
	if len(details.Blobs) > 0 {
		b.WriteString("\r\nBLOBs/Fotos (SHA-256)\r\n")
		for _, name := range sortedMapKeys(details.Blobs) {
			prefix := ""
			if containsStringFold(conflict.DifferingColumns, name) {
				prefix = "[DIFERENTE] "
			} else if containsStringFold(conflict.InconclusiveColumns, name) {
				prefix = "[INCONCLUSIVO] "
			}
			b.WriteString(prefix + name + ": " + details.Blobs[name] + "\r\n")
		}
	}
	return b.String()
}

func duplicateComparisonFilePath(reportPath string, conflict restoreDuplicateConflict, round int) string {
	return filepath.Join(filepath.Dir(reportPath), fmt.Sprintf("COMPARACAO_CONFLITO_%s_%02d.txt", sanitizeFilePart(conflict.Relation), round))
}

func buildDuplicateResolutionView(ctx context.Context, root, version, port, user, password, dbPath, reportPath string, conflict *restoreDuplicateConflict, round int) (duplicateResolutionView, error) {
	var view duplicateResolutionView
	if conflict == nil {
		return view, errors.New("conflito não informado")
	}
	meta, err := loadRelationMetadataAt(ctx, root, version, port, user, password, dbPath, conflict.Relation)
	if err != nil {
		return view, err
	}
	first, err := loadDuplicateOccurrenceDetails(ctx, root, version, port, user, password, dbPath, meta, *conflict, 1)
	if err != nil {
		return view, err
	}
	second, err := loadDuplicateOccurrenceDetails(ctx, root, version, port, user, password, dbPath, meta, *conflict, 2)
	if err != nil {
		return view, err
	}
	applyBlobComparison(conflict, first, second, meta, version)
	cols := duplicateSummaryColumns(meta, *conflict)
	view.Record1Text = formatDuplicateOccurrence(first, cols, *conflict)
	view.Record2Text = formatDuplicateOccurrence(second, cols, *conflict)
	var summary strings.Builder
	summary.WriteString("Tabela: " + conflict.Relation + "\r\n")
	summary.WriteString("Chave: " + duplicateConflictKeyLabel(*conflict) + "\r\n")
	summary.WriteString(fmt.Sprintf("Ocorrências encontradas: %d\r\n", conflict.Occurrences))
	if len(conflict.DifferingColumns) > 0 {
		summary.WriteString("Campos diferentes: " + strings.Join(conflict.DifferingColumns, ", ") + "\r\n")
	}
	if len(conflict.InconclusiveColumns) > 0 {
		summary.WriteString("Ainda inconclusivo: " + strings.Join(conflict.InconclusiveColumns, ", ") + "\r\n")
	}
	if len(conflict.BlobComparison) > 0 {
		summary.WriteString("\r\nComparação de BLOBs/Fotos:\r\n")
		for _, note := range conflict.BlobComparison {
			summary.WriteString("- " + note + "\r\n")
		}
	}
	view.SummaryText = summary.String()
	view.DetailsText = "Escolha explicitamente qual registro deve permanecer na CÓPIA DE TRABALHO.\r\nO banco original e a cópia de inspeção permanecem preservados.\r\nApós a escolha, o Toolbox continua automaticamente: índices → validação → novo backup → restore de teste."
	view.FilePath = duplicateComparisonFilePath(reportPath, *conflict, round)
	full := "FIREBIRD TOOLBOX - COMPARAÇÃO DE CONFLITO\r\nVersão: " + appVersion + "\r\n\r\n" + view.SummaryText + "\r\n" + view.Record1Text + "\r\n" + view.Record2Text + "\r\n" + view.DetailsText + "\r\n"
	if err := os.WriteFile(view.FilePath, []byte(full), 0600); err != nil {
		return view, err
	}
	conflict.ComparisonFile = view.FilePath
	return view, nil
}

func duplicateCleanupKeepOccurrenceScript(conflict restoreDuplicateConflict, keepOrdinal int) (string, error) {
	if keepOrdinal < 1 {
		return "", errors.New("ocorrência escolhida inválida")
	}
	where, err := duplicateKeyPredicate(conflict, "T")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("SET BAIL ON;\nSET TERM ^;\nEXECUTE BLOCK AS\n")
	b.WriteString("DECLARE VARIABLE K CHAR(8) CHARACTER SET OCTETS;\nDECLARE VARIABLE KEEP_K CHAR(8) CHARACTER SET OCTETS;\nDECLARE VARIABLE N INTEGER;\nBEGIN\n  N = 0;\n")
	b.WriteString("  FOR SELECT T.RDB$DB_KEY FROM " + sqlIdentifier(conflict.Relation) + " T WHERE " + where + " PLAN (T NATURAL) INTO :K DO\n  BEGIN\n")
	b.WriteString(fmt.Sprintf("    N = N + 1;\n    IF (N = %d) THEN KEEP_K = K;\n", keepOrdinal))
	b.WriteString("  END\n")
	b.WriteString("  FOR SELECT T.RDB$DB_KEY FROM " + sqlIdentifier(conflict.Relation) + " T WHERE " + where + " PLAN (T NATURAL) INTO :K DO\n  BEGIN\n")
	b.WriteString("    IF (KEEP_K IS NOT NULL AND K <> KEEP_K) THEN DELETE FROM " + sqlIdentifier(conflict.Relation) + " WHERE RDB$DB_KEY = :K;\n  END\nEND^\nSET TERM ;^\nCOMMIT;\nQUIT;\n")
	return b.String(), nil
}

func duplicateCleanupScript(conflict restoreDuplicateConflict) (string, error) {
	return duplicateCleanupKeepOccurrenceScript(conflict, 1)
}

func constraintIndexName(ctx context.Context, root, version, port, user, password, dbPath string, conflict restoreDuplicateConflict) (string, error) {
	script := "SET HEADING OFF;\nSET LIST OFF;\nSET COUNT OFF;\nSELECT 'FTB_CINDEX|' || TRIM(RDB$INDEX_NAME) FROM RDB$RELATION_CONSTRAINTS WHERE RDB$CONSTRAINT_NAME = " + sqlStringLiteral(conflict.Constraint) + " AND RDB$RELATION_NAME = " + sqlStringLiteral(conflict.Relation) + ";\nQUIT;\n"
	out, err := runISQLScript(ctx, root, version, port, user, password, dbPath, "Localizando índice da constraint "+conflict.Constraint+"...", script)
	if err != nil {
		return "", err
	}
	lines := markerLines(out, "FTB_CINDEX|")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "", fmt.Errorf("não foi possível localizar o índice da constraint %s", conflict.Constraint)
	}
	return strings.TrimSpace(lines[0]), nil
}

func bringDatabaseOnline(ctx context.Context, root, version, port, user, password, dbPath string) error {
	gfix := firebirdExe(root, version, "gfix.exe")
	if !fileExists(gfix) {
		return fmt.Errorf("não encontrei gfix.exe do Firebird %s", version)
	}
	_, err := runRecoveryCapture(ctx, "Colocando banco parcial online somente para inspeção e saneamento...", gfix, "-user", user, "-pass", password, "-online", databaseConnection(port, dbPath))
	return err
}

func backupDatabaseWithEngine(ctx context.Context, root, version, port, user, password, dbPath, backup, label string) error {
	gbak := firebirdExe(root, version, "gbak.exe")
	if !fileExists(gbak) {
		return fmt.Errorf("não encontrei gbak.exe do Firebird %s", version)
	}
	_ = os.Remove(backup)
	_, err := runRecoveryCapture(ctx, label, gbak, "-b", "-v", "-user", user, "-pass", password, databaseConnection(port, dbPath), backup)
	if err != nil {
		return err
	}
	if !fileExists(backup) {
		return fmt.Errorf("o backup saneado não foi criado: %s", backup)
	}
	return nil
}

func failedRestoreSnapshotPath(testPath string, round int) string {
	ext := filepath.Ext(testPath)
	base := strings.TrimSuffix(testPath, ext)
	if ext == "" {
		ext = ".RED"
	}
	return fmt.Sprintf("%s_RESTORE_FALHO_%02d%s", base, round, ext)
}

func conflictSnapshotPath(testPath string, round int) string {
	ext := filepath.Ext(testPath)
	base := strings.TrimSuffix(testPath, ext)
	if ext == "" {
		ext = ".RED"
	}
	return fmt.Sprintf("%s_CONFLITO_ORIGINAL_%02d%s", base, round, ext)
}

func conflictRepairedPath(testPath string, round int) string {
	ext := filepath.Ext(testPath)
	base := strings.TrimSuffix(testPath, ext)
	if ext == "" {
		ext = ".RED"
	}
	return fmt.Sprintf("%s_CONFLITO_SANEADO_%02d%s", base, round, ext)
}

func deduplicatedBackupPath(backup string, round int) string {
	ext := filepath.Ext(backup)
	base := strings.TrimSuffix(backup, ext)
	if ext == "" {
		ext = ".BAK"
	}
	return fmt.Sprintf("%s_SEM_DUPLICIDADES_%02d%s", base, round, ext)
}

func conflictLoopBackupPath(backup string, round int) string {
	ext := filepath.Ext(backup)
	base := strings.TrimSuffix(backup, ext)
	if ext == "" {
		ext = ".BAK"
	}
	return fmt.Sprintf("%s_CONFLITOS_%02d%s", base, round, ext)
}

func cleanRelationBaseBackupPath(backup, relation string, round int) string {
	ext := filepath.Ext(backup)
	base := strings.TrimSuffix(backup, ext)
	if ext == "" {
		ext = ".BAK"
	}
	return fmt.Sprintf("%s_REBUILD_%s_BASE_%02d%s", base, sanitizeFilePart(relation), round, ext)
}

func cleanRelationBackupPath(backup, relation string, round int) string {
	ext := filepath.Ext(backup)
	base := strings.TrimSuffix(backup, ext)
	if ext == "" {
		ext = ".BAK"
	}
	return fmt.Sprintf("%s_REBUILD_%s_%02d%s", base, sanitizeFilePart(relation), round, ext)
}

func cleanRelationStagePath(testPath, relation string, round int) string {
	ext := filepath.Ext(testPath)
	base := strings.TrimSuffix(testPath, ext)
	if ext == "" {
		ext = ".RED"
	}
	return fmt.Sprintf("%s_REBUILD_%s_%02d%s", base, sanitizeFilePart(relation), round, ext)
}

func setRelationTriggersActive(ctx context.Context, root, version, port, user, password, dbPath string, triggers []string, active bool) error {
	if len(triggers) == 0 {
		return nil
	}
	state := "INACTIVE"
	label := "Desativando triggers da relação no banco de reconstrução..."
	if active {
		state = "ACTIVE"
		label = "Reativando triggers da relação no banco de reconstrução..."
	}
	var b strings.Builder
	b.WriteString("SET BAIL ON;\n")
	for _, name := range triggers {
		b.WriteString("ALTER TRIGGER " + sqlIdentifier(name) + " " + state + ";\n")
	}
	b.WriteString("COMMIT;\nQUIT;\n")
	_, err := runISQLScript(ctx, root, version, port, user, password, dbPath, label, b.String())
	return err
}

func backupWithoutRelationData(ctx context.Context, root, version, port, user, password, dbPath, relation, backup string) error {
	if versionNumber(version) < 3.0 {
		return fmt.Errorf("a reconstrução limpa automática exige Firebird 3.0 ou superior para usar -skip_data")
	}
	gbak := firebirdExe(root, version, "gbak.exe")
	if !fileExists(gbak) {
		return fmt.Errorf("não encontrei gbak.exe do Firebird %s", version)
	}
	_ = os.Remove(backup)
	skipPattern := "^" + regexp.QuoteMeta(relation) + "$"
	args := []string{"-b", "-v", "-skip_data", skipPattern, "-user", user, "-pass", password, databaseConnection(port, dbPath), backup}
	_, err := runRecoveryCapture(ctx, "Criando base limpa sem os dados de "+relation+"...", gbak, args...)
	if err != nil {
		return err
	}
	if !fileExists(backup) {
		return fmt.Errorf("o backup-base sem %s não foi criado", relation)
	}
	return nil
}

func rebuildRelationCleanBackup(ctx context.Context, c workflowConfig, sourceDB, currentBackup string, report *recoveryReport, relation, version, root, port string, round int) (string, string, error) {
	if strings.TrimSpace(relation) == "" {
		return "", "", errors.New("relação não informada para reconstrução limpa")
	}
	if versionNumber(version) < 3.0 {
		return "", "", fmt.Errorf("%s continuou com índice inativo e a reconstrução limpa automática requer Firebird 3.0 ou superior", relation)
	}
	meta, err := loadRelationMetadataAt(ctx, root, version, port, c.User, c.Password, sourceDB, relation)
	if err != nil {
		return "", "", fmt.Errorf("não foi possível ler os metadados de %s para reconstrução limpa: %w", relation, err)
	}
	if hasUnsupportedArray(meta) {
		return "", "", fmt.Errorf("%s possui coluna ARRAY; reconstrução limpa automática não é segura", relation)
	}
	sourceCount, countErr := countRelationRows(ctx, root, version, port, c.User, c.Password, sourceDB, relation)
	if countErr != nil {
		return "", "", fmt.Errorf("não foi possível contar %s antes da reconstrução limpa: %w", relation, countErr)
	}

	baseBackup := cleanRelationBaseBackupPath(currentBackup, relation, round)
	if err := backupWithoutRelationData(ctx, root, version, port, c.User, c.Password, sourceDB, relation, baseBackup); err != nil {
		return "", "", fmt.Errorf("falha ao criar base para reconstrução limpa de %s: %w", relation, err)
	}
	stagePath := cleanRelationStagePath(sourceDB, relation, round)
	_ = os.Remove(stagePath)
	if err := restoreBackupWithInactiveIndexes(ctx, root, version, port, c.User, c.Password, baseBackup, stagePath); err != nil {
		return "", "", fmt.Errorf("não foi possível restaurar a base limpa de %s: %w", relation, err)
	}
	if err := setRelationTriggersActive(ctx, root, version, port, c.User, c.Password, stagePath, meta.ActiveTriggers, false); err != nil {
		return "", stagePath, fmt.Errorf("não foi possível desativar os triggers de %s no banco limpo: %w", relation, err)
	}
	triggersNeedRestore := true
	defer func() {
		if triggersNeedRestore {
			_ = setRelationTriggersActive(context.Background(), root, version, port, c.User, c.Password, stagePath, meta.ActiveTriggers, true)
		}
	}()

	copyScript, err := copyRelationBlockToDataSource(meta, c.User, c.Password, externalDataSourceAt(root, version, port, stagePath), "", true)
	if err != nil {
		return "", stagePath, err
	}
	out, copyErr := runISQLScript(ctx, root, version, port, c.User, c.Password, sourceDB, "Reconstruindo "+relation+" em estrutura limpa por leitura NATURAL...", copyScript)
	if copyErr != nil {
		if conflicts := parseRestoreDuplicateConflicts(out); len(conflicts) > 0 {
			first := conflicts[0]
			return "", stagePath, fmt.Errorf("a reconstrução limpa encontrou nova duplicidade em %s (%s)", first.Relation, duplicateConflictKeyLabel(first))
		}
		return "", stagePath, fmt.Errorf("falha ao copiar os registros de %s para a estrutura limpa: %w", relation, copyErr)
	}
	recovered := parseRecoveredRows(out)
	if recovered != sourceCount {
		return "", stagePath, fmt.Errorf("reconstrução limpa de %s copiou %d de %d registros; banco preservado para inspeção", relation, recovered, sourceCount)
	}
	if err := setRelationTriggersActive(ctx, root, version, port, c.User, c.Password, stagePath, meta.ActiveTriggers, true); err != nil {
		return "", stagePath, fmt.Errorf("dados de %s foram copiados, mas os triggers não puderam ser reativados: %w", relation, err)
	}
	triggersNeedRestore = false

	remaining, conflicts, activateErr := activateInactiveIndexesDetectingConflicts(ctx, root, version, port, c.User, c.Password, stagePath)
	if activateErr != nil {
		return "", stagePath, fmt.Errorf("falha ao reconstruir índices após recriar %s: %w", relation, activateErr)
	}
	if len(conflicts) > 0 {
		return "", stagePath, fmt.Errorf("a estrutura limpa de %s ainda apresentou conflito de chave %s", relation, duplicateConflictKeyLabel(conflicts[0]))
	}
	if len(remaining) > 0 {
		var names []string
		for _, item := range remaining {
			names = append(names, item.IndexName)
		}
		return "", stagePath, fmt.Errorf("a estrutura limpa de %s ainda possui índice(s) inativo(s): %s", relation, strings.Join(names, ", "))
	}

	stageCount, err := countRelationRows(ctx, root, version, port, c.User, c.Password, stagePath, relation)
	if err != nil || stageCount != sourceCount {
		return "", stagePath, fmt.Errorf("a conferência final de %s falhou após reconstrução limpa: origem=%d destino=%d", relation, sourceCount, stageCount)
	}
	gfix := firebirdExe(root, version, "gfix.exe")
	if fileExists(gfix) {
		args := []string{"-user", c.User, "-pass", c.Password, "-v", "-full"}
		if versionNumber(version) >= 2.1 {
			args = append(args, "-no_update")
		}
		args = append(args, databaseConnection(port, stagePath))
		validationOutput, validationErr := runRecoveryCapture(ctx, "Validando banco após reconstrução limpa de "+relation+"...", gfix, args...)
		if validationErr != nil || toolOutputHasOperationalFailure(validationOutput) || parseRecoveryStats(validationOutput).HasCorruption() {
			return "", stagePath, fmt.Errorf("o banco reconstruído após recriar %s não passou na validação estrutural", relation)
		}
	}

	finalBackup := cleanRelationBackupPath(currentBackup, relation, round)
	if err := backupDatabaseWithEngine(ctx, root, version, port, c.User, c.Password, stagePath, finalBackup, "Gerando backup após reconstrução limpa de "+relation+"..."); err != nil {
		return "", stagePath, fmt.Errorf("não foi possível gerar backup da estrutura limpa de %s: %w", relation, err)
	}
	report.CleanRebuildAttempted = true
	report.CleanRebuildRelation = relation
	report.CleanRebuildStage = stagePath
	report.CleanRebuildBackup = finalBackup
	report.action(fmt.Sprintf("Relação %s reconstruída em estrutura limpa; %d registro(s) copiado(s) por leitura NATURAL e todos os índices reativados", relation, sourceCount))
	appendLog(fmt.Sprintf("✓ %s reconstruída em estrutura limpa com %d registro(s); novo restore de teste será executado.", relation, sourceCount))
	return finalBackup, stagePath, nil
}

func writeDuplicateConflictReport(path, snapshot string, conflicts []restoreDuplicateConflict, exports []string) error {
	var b strings.Builder
	b.WriteString("FIREBIRD TOOLBOX - CONFLITOS DE CHAVE ÚNICA / PRIMÁRIA\r\n")
	b.WriteString("Versão: " + appVersion + "\r\n")
	b.WriteString("Banco de inspeção preservado: " + snapshot + "\r\n")
	for i, conflict := range conflicts {
		b.WriteString(fmt.Sprintf("\r\nCONFLITO %d\r\n", i+1))
		b.WriteString("Tabela: " + conflict.Relation + "\r\n")
		b.WriteString("Constraint: " + conflict.Constraint + "\r\n")
		if conflict.IndexName != "" {
			b.WriteString("Índice: " + conflict.IndexName + "\r\n")
		}
		b.WriteString("Chave: " + duplicateConflictKeyLabel(conflict) + "\r\n")
		b.WriteString(fmt.Sprintf("Ocorrências encontradas: %d\r\n", conflict.Occurrences))
		if conflict.Identical {
			b.WriteString("Comparação: ocorrências logicamente idênticas nas colunas comparáveis.\r\n")
		} else if len(conflict.DifferingColumns) > 0 {
			b.WriteString("Campos diferentes: " + strings.Join(conflict.DifferingColumns, ", ") + "\r\n")
		} else if conflict.ComparisonInconclusive {
			b.WriteString("Comparação: inconclusiva.\r\n")
		}
		if len(conflict.InconclusiveColumns) > 0 {
			b.WriteString("Colunas não comparadas automaticamente: " + strings.Join(conflict.InconclusiveColumns, ", ") + "\r\n")
		}
		if len(conflict.BlobComparison) > 0 {
			b.WriteString("Comparação de BLOBs/Fotos:\r\n")
			for _, note := range conflict.BlobComparison {
				b.WriteString("- " + note + "\r\n")
			}
		}
		if conflict.ChosenOccurrence > 0 {
			b.WriteString(fmt.Sprintf("Ocorrência escolhida para permanecer na cópia: REGISTRO %d\r\n", conflict.ChosenOccurrence))
		}
		if conflict.ComparisonFile != "" {
			b.WriteString("Arquivo de comparação: " + conflict.ComparisonFile + "\r\n")
		}
		if conflict.RowsRemoved > 0 {
			b.WriteString(fmt.Sprintf("Ocorrências extras removidas na CÓPIA DE TRABALHO: %d\r\n", conflict.RowsRemoved))
		}
		if i < len(exports) && strings.TrimSpace(exports[i]) != "" {
			b.WriteString("\r\nDADOS EXPORTADOS PARA INSPEÇÃO\r\n")
			b.WriteString(strings.ReplaceAll(exports[i], "\n", "\r\n"))
			b.WriteString("\r\n")
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0600)
}

func repairRestoreDuplicateConflicts(ctx context.Context, c workflowConfig, backup, testPath string, report *recoveryReport, conflicts []restoreDuplicateConflict, version, root, port string, round int) (string, error) {
	if len(conflicts) == 0 {
		return "", errors.New("nenhum conflito de chave foi informado")
	}
	if versionNumber(version) < 2.0 {
		return "", fmt.Errorf("a recuperação automática de duplicidades exige Firebird 2.0 ou superior; conflito detectado em %s", conflicts[0].Relation)
	}
	failedRestore := ""
	if fileExists(testPath) {
		failedRestore = failedRestoreSnapshotPath(testPath, round)
		_ = os.Remove(failedRestore)
		if err := os.Rename(testPath, failedRestore); err != nil {
			if _, _, copyErr := fileCopyWithSHA256(testPath, failedRestore); copyErr != nil {
				return "", fmt.Errorf("não foi possível preservar o restore originalmente falho: %w", err)
			}
			_ = os.Remove(testPath)
		}
	}
	report.DuplicateConflictDetected = true
	if failedRestore != "" {
		report.DuplicateFailedRestore = failedRestore
		report.action("Restore que falhou na ativação dos índices preservado para auditoria")
	}

	if err := restoreBackupWithInactiveIndexes(ctx, root, version, port, c.User, c.Password, backup, testPath); err != nil {
		report.DuplicateConflictSource = failedRestore
		return "", fmt.Errorf("o restore normal identificou duplicidade, mas a nova restauração com -inactive também falhou: %w", err)
	}
	report.action("Backup restaurado novamente com -inactive para preservar todos os registros e permitir saneamento controlado")

	snapshot := conflictSnapshotPath(testPath, round)
	_ = os.Remove(snapshot)
	if _, _, err := fileCopyWithSHA256(testPath, snapshot); err != nil {
		return "", fmt.Errorf("não foi possível preservar o banco de inspeção antes do saneamento: %w", err)
	}
	report.DuplicateConflictSource = snapshot
	report.action("Banco completo com todos os índices inativos e todas as ocorrências conflitantes preservado antes de qualquer remoção")

	conflictReport := filepath.Join(filepath.Dir(report.ReportPath), "CONFLITOS_DUPLICIDADE.txt")
	report.DuplicateConflictReport = conflictReport
	exports := make([]string, 0, len(conflicts))
	analyzed := make([]restoreDuplicateConflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		indexName, idxErr := constraintIndexName(ctx, root, version, port, c.User, c.Password, testPath, conflict)
		if idxErr == nil {
			conflict.IndexName = indexName
		}
		checked, export, err := analyzeDuplicateConflict(ctx, root, version, port, c.User, c.Password, testPath, conflict)
		if err != nil {
			return "", fmt.Errorf("falha ao analisar chave duplicada %s em %s: %w", duplicateConflictKeyLabel(conflict), conflict.Relation, err)
		}
		if checked.Occurrences < 2 {
			return "", fmt.Errorf("o restore reportou duplicidade em %s, mas a cópia parcial contém apenas %d ocorrência(s) da chave %s", checked.Relation, checked.Occurrences, duplicateConflictKeyLabel(checked))
		}
		if checked.Occurrences > 2 {
			report.DuplicateConflictDetected = true
			report.warning(fmt.Sprintf("A chave %s em %s possui %d ocorrências; o resolvedor visual desta versão exige exatamente duas para evitar remoção ambígua", duplicateConflictKeyLabel(checked), checked.Relation, checked.Occurrences))
			return "", fmt.Errorf("conflito com %d ocorrências requer inspeção manual em %s (%s); nenhuma ocorrência foi removida", checked.Occurrences, checked.Relation, duplicateConflictKeyLabel(checked))
		}
		if checked.IndexName == "" {
			checked.IndexName = conflict.IndexName
		}
		analyzed = append(analyzed, checked)
		exports = append(exports, export)
	}
	_ = writeDuplicateConflictReport(conflictReport, snapshot, analyzed, exports)

	for i := range analyzed {
		conflict := &analyzed[i]
		keepOrdinal := 1

		if !conflict.Identical {
			view, viewErr := buildDuplicateResolutionView(ctx, root, version, port, c.User, c.Password, testPath, report.ReportPath, conflict, round+i)
			if viewErr != nil {
				report.warning("Não foi possível montar a comparação visual completa: " + viewErr.Error())
				view = duplicateResolutionView{
					SummaryText: "Tabela: " + conflict.Relation + "\r\nChave: " + duplicateConflictKeyLabel(*conflict),
					Record1Text: "Registro 1 preservado no banco de inspeção.",
					Record2Text: "Registro 2 preservado no banco de inspeção.",
					DetailsText: "A comparação detalhada falhou. Preserve os registros e revise CONFLITOS_DUPLICIDADE.txt antes de decidir.",
					FilePath:    conflictReport,
				}
			}

			if conflict.Identical {
				report.action("Duplicidade estrutural idêntica confirmada após comparar BLOBs/Fotos em " + conflict.Relation + "; a primeira ocorrência será preservada")
			} else {
				decision := showDuplicateConflictResolver(*conflict, view)
				switch decision {
				case conflictResolutionKeep1:
					keepOrdinal = 1
				case conflictResolutionKeep2:
					keepOrdinal = 2
				default:
					report.DuplicateConflicts = append(report.DuplicateConflicts, analyzed...)
					report.warning("Conflito de chave preservado sem escolha: " + conflict.Relation + " | " + duplicateConflictKeyLabel(*conflict))
					report.FinalState = "CONFLITO DE DADOS AGUARDANDO DECISÃO; BANCO PARCIAL PRESERVADO"
					_ = writeDuplicateConflictReport(conflictReport, snapshot, analyzed, exports)
					return "", fmt.Errorf("conflito de dados requer decisão manual em %s (%s); nenhuma ocorrência foi removida", conflict.Relation, duplicateConflictKeyLabel(*conflict))
				}
				conflict.ChosenOccurrence = keepOrdinal
				conflict.UserAcceptedDataChoice = true
				report.PartialRecovery = true
				report.warning(fmt.Sprintf("Usuário escolheu explicitamente manter o REGISTRO %d de %s para a chave %s; as outras ocorrências serão removidas somente na cópia de trabalho", keepOrdinal, conflict.Relation, duplicateConflictKeyLabel(*conflict)))
			}
		} else {
			report.action("Duplicidade estrutural idêntica detectada em " + conflict.Relation + "; a primeira ocorrência será preservada")
		}

		cleanup, err := duplicateCleanupKeepOccurrenceScript(*conflict, keepOrdinal)
		if err != nil {
			return "", err
		}
		if _, err := runISQLScript(ctx, root, version, port, c.User, c.Password, testPath, fmt.Sprintf("Mantendo o REGISTRO %d e removendo apenas ocorrências extras da chave duplicada em %s...", keepOrdinal, conflict.Relation), cleanup); err != nil {
			return "", fmt.Errorf("não foi possível remover a duplicidade de %s: %w", conflict.Relation, err)
		}
		remaining, err := duplicateOccurrenceCount(ctx, root, version, port, c.User, c.Password, testPath, *conflict)
		if err != nil {
			return "", err
		}
		if remaining != 1 {
			return "", fmt.Errorf("saneamento da chave %s em %s terminou com %d ocorrências; esperado: 1", duplicateConflictKeyLabel(*conflict), conflict.Relation, remaining)
		}
		conflict.RowsRemoved = conflict.Occurrences - 1
		report.DuplicateRowsRemoved += conflict.RowsRemoved
		report.action(fmt.Sprintf("Conflito %s / %s saneado na cópia de trabalho; REGISTRO %d preservado e %d ocorrência(s) extra(s) removida(s)", conflict.Relation, duplicateConflictKeyLabel(*conflict), keepOrdinal, conflict.RowsRemoved))
	}
	report.DuplicateConflicts = append(report.DuplicateConflicts, analyzed...)
	_ = writeDuplicateConflictReport(conflictReport, snapshot, report.DuplicateConflicts, nil)

	remainingIndexes, nextConflicts, activationErr := activateInactiveIndexesDetectingConflicts(ctx, root, version, port, c.User, c.Password, testPath)
	if activationErr != nil {
		return "", fmt.Errorf("falha ao reconstruir os índices após o saneamento: %w", activationErr)
	}
	if len(nextConflicts) > 0 {
		if round >= 24 {
			return "", fmt.Errorf("o limite de 24 rodadas de conflitos foi atingido; banco preservado para inspeção")
		}
		first := nextConflicts[0]
		appendLog("Novo conflito encontrado ao reconstruir os índices: " + first.Relation + " | " + duplicateConflictKeyLabel(first))
		report.action("Após resolver um conflito, a reconstrução dos índices encontrou automaticamente outra chave duplicada; o saneamento continuará sem abandonar o fluxo")
		loopBackup := conflictLoopBackupPath(backup, round)
		if err := backupDatabaseWithEngine(ctx, root, version, port, c.User, c.Password, testPath, loopBackup, "Preservando estado saneado antes de resolver o próximo conflito..."); err != nil {
			return "", fmt.Errorf("foi encontrado outro conflito, mas não foi possível preservar o estado intermediário: %w", err)
		}
		return repairRestoreDuplicateConflicts(ctx, c, loopBackup, testPath, report, nextConflicts, version, root, port, round+1)
	}
	if len(remainingIndexes) > 0 {
		relations := uniqueRelationsFromInactive(remainingIndexes)
		if len(relations) == 0 {
			var names []string
			for _, item := range remainingIndexes {
				names = append(names, item.IndexName)
			}
			return "", fmt.Errorf("as duplicidades foram tratadas, mas %d índice(s) permaneceram inativos sem relação de usuário reconstruível: %s", len(remainingIndexes), strings.Join(names, ", "))
		}
		relation := relations[0]
		report.CleanRebuildAttempted = true
		report.CleanRebuildRelation = relation
		report.warning("Nenhuma nova chave duplicada foi encontrada, mas índices de " + relation + " continuaram inativos; iniciando reconstrução limpa da relação em banco temporário")
		appendLog("Índices ainda inativos sem nova duplicidade detectável. Reconstruindo " + relation + " em estrutura limpa...")
		rebuiltBackup, stagePath, rebuildErr := rebuildRelationCleanBackup(ctx, c, testPath, backup, report, relation, version, root, port, round)
		if rebuildErr != nil {
			report.CleanRebuildStage = stagePath
			return "", fmt.Errorf("a reconstrução limpa de %s não foi concluída: %w", relation, rebuildErr)
		}
		report.DuplicateConflictWork = stagePath
		report.DuplicateRepairApplied = true
		report.DuplicateRepairedBackup = rebuiltBackup
		report.action("Fallback de reconstrução limpa concluído; o novo backup será submetido a restore completo e gates de saúde")
		return rebuiltBackup, nil
	}
	report.action("Todos os índices de usuário do restore com -inactive foram reconstruídos e reativados após o saneamento")

	gfix := firebirdExe(root, version, "gfix.exe")
	if fileExists(gfix) {
		args := []string{"-user", c.User, "-pass", c.Password, "-v", "-full"}
		if versionNumber(version) >= 2.1 {
			args = append(args, "-no_update")
		}
		args = append(args, databaseConnection(port, testPath))
		validationOutput, validationErr := runRecoveryCapture(ctx, "Validando banco saneado após resolver duplicidades...", gfix, args...)
		if validationErr != nil || toolOutputHasOperationalFailure(validationOutput) {
			return "", fmt.Errorf("o banco saneado não passou na validação após resolver as duplicidades")
		}
	}

	repairedBackup := deduplicatedBackupPath(backup, round)
	if err := backupDatabaseWithEngine(ctx, root, version, port, c.User, c.Password, testPath, repairedBackup, "Gerando backup após saneamento das chaves duplicadas..."); err != nil {
		return "", fmt.Errorf("falha ao gerar backup após saneamento das duplicidades: %w", err)
	}
	repairedDB := conflictRepairedPath(testPath, round)
	_ = os.Remove(repairedDB)
	if err := os.Rename(testPath, repairedDB); err == nil {
		report.DuplicateConflictWork = repairedDB
	} else {
		report.warning("O banco saneado permaneceu no caminho temporário porque não foi possível renomeá-lo: " + err.Error())
		report.DuplicateConflictWork = testPath
	}
	report.DuplicateRepairApplied = true
	report.DuplicateRepairedBackup = repairedBackup
	report.action("Novo backup gerado após resolver conflitos de chave; será submetido a um novo restore de teste completo")
	return repairedBackup, nil
}

type restoreTestResult struct {
	Verified bool
	Partial  bool
	// NoValidity marca o restore que so passou com -n, ou seja, sem as
	// constraints de validade e as FOREIGN KEY. O banco nao e equivalente ao
	// original e nunca pode ser liberado para producao sem revisao manual.
	NoValidity bool
	Path       string
	Backup     string
	Header     databaseHeader
}

func verifyBackupByRestore(ctx context.Context, c workflowConfig, backup, testPath string, report *recoveryReport) (restoreTestResult, error) {
	return verifyBackupByRestoreDepth(ctx, c, backup, testPath, report, 1)
}

func verifyBackupByRestoreDepth(ctx context.Context, c workflowConfig, backup, testPath string, report *recoveryReport, round int) (restoreTestResult, error) {
	version, root, port := testRestoreEngine(c)
	gbak := firebirdExe(root, version, "gbak.exe")
	if !fileExists(gbak) {
		return restoreTestResult{}, fmt.Errorf("não encontrei gbak.exe do Firebird %s para o restore de teste", version)
	}
	resolvedPort := port
	if supportsServiceManagerRestore(version) {
		p, err := verifyFirebirdEndpoint(ctx, "do restore de teste", version, root, "", port)
		if err != nil {
			return restoreTestResult{}, err
		}
		resolvedPort = p
	}
	_ = os.Remove(testPath)
	setProgress(76, "Testando o backup...")
	output, err := runRecoveryCapture(ctx, "Testando backup com uma restauração temporária...", gbak, restoreTestArgs(version, resolvedPort, c.User, c.Password, backup, testPath, false, false)...)
	if err == nil && fileExists(testPath) {
		if report != nil && len(report.IndexesDisabled) > 0 && !report.DuplicateRepairApplied {
			if _, activeErr := runISQLScript(ctx, root, version, resolvedPort, c.User, c.Password, testPath, "Reativando índices no restore de teste...", buildAlterIndexesScript(report.IndexesDisabled, true)); activeErr != nil {
				report.warning("O restore de teste foi criado, mas a reativação dos índices falhou")
				report.TestRestore = testPath
				return restoreTestResult{Partial: true, Path: testPath}, fmt.Errorf("não foi possível reconstruir/reativar os índices após o restore de teste: %w", activeErr)
			}
			report.action("Índices temporariamente inativos na cópia foram reconstruídos/reativados no restore de teste")
		}
		report.action("Restore de teste completo concluído com sucesso no Firebird " + version)
		result := restoreTestResult{Verified: true, Path: testPath, Backup: backup}
		gstat := firebirdExe(root, version, "gstat.exe")
		if fileExists(gstat) {
			headerOutput, headerErr := runCommandCapture(ctx, gstat, "-h", testPath)
			if headerErr == nil {
				result.Header = parseDatabaseHeader(headerOutput)
			}
		}
		// Preserve o restore verificado até os gates de pós-recuperação. Ele é a
		// referência independente para comparar metadados e contagens do destino
		// final. Só é removido depois que todos os gates forem aprovados.
		report.TestRestore = testPath
		report.action("Restore de teste preservado temporariamente para comparação pós-recuperação")
		return result, nil
	}

	if output != "" && !detailedLog {
		appendLog("O restore de teste normal falhou. Últimas mensagens:")
		appendLog(tailText(output, 12))
	}

	duplicateConflicts := parseRestoreDuplicateConflicts(output)
	if len(duplicateConflicts) > 0 {
		first := duplicateConflicts[0]
		appendLog("Conflito de chave detectado no restore: " + first.Relation + " | " + duplicateConflictKeyLabel(first))
		report.warning(fmt.Sprintf("Restore detectou %d conflito(s) de chave primária/única; -one_at_a_time não será usado porque não resolve duplicidades", len(duplicateConflicts)))
		if round > 3 {
			report.FinalState = "CONFLITOS DE CHAVE PERSISTEM APÓS TENTATIVAS DE SANEAMENTO"
			return restoreTestResult{Partial: fileExists(testPath), Path: testPath, Backup: backup}, fmt.Errorf("conflitos de chave persistem após %d rodadas de saneamento", round-1)
		}
		repairedBackup, repairErr := repairRestoreDuplicateConflicts(ctx, c, backup, testPath, report, duplicateConflicts, version, root, resolvedPort, round)
		if repairErr != nil {
			path := report.DuplicateConflictSource
			if path == "" {
				path = testPath
			}
			return restoreTestResult{Partial: true, Path: path, Backup: backup}, repairErr
		}
		appendLog("✓ Conflitos de chave tratados na cópia. Repetindo restore de teste completo...")
		return verifyBackupByRestoreDepth(ctx, c, repairedBackup, testPath, report, round+1)
	}

	// Falha ao recriar FOREIGN KEY: linhas-pai perdidas nas paginas corrompidas
	// deixaram filhos apontando para PKs inexistentes. O -one_at_a_time nao
	// resolve isso, entao tentamos -n antes dele.
	if gbakLooksLikeOrphanForeignKey(output) {
		report.OrphanFKDetected = true
		appendLog("Restore falhou ao recriar uma FOREIGN KEY: há linhas órfãs apontando para chaves que se perderam na corrupção.")
		report.warning("Restore detectou integridade referencial quebrada; -one_at_a_time não resolve porque a falha é na criação da constraint, não nos dados")
		if messageBox("Integridade referencial quebrada", "O restore falhou ao recriar uma FOREIGN KEY:\n\nNon-existent PRIMARY or UNIQUE KEY specified for FOREIGN KEY\n\nOs registros perdidos na corrupção eram linhas-pai, e os filhos sobreviveram apontando para chaves que não existem mais.\n\nDeseja tentar o restore com -n (-no_validity)?\n\nATENÇÃO: o banco restaurado ficará SEM as FOREIGN KEY e SEM as constraints CHECK. Elas terão de ser recriadas manualmente depois de sanear as linhas órfãs.\n\nO DDL das constraints originais será extraído automaticamente antes do restore e gravado na pasta RECOVERY_... em "+constraintsScriptName+". As consultas que listam as linhas órfãs ficarão em "+orphanRowsScriptName+", na mesma pasta.", MB_YESNO|MB_ICONWARNING) == IDYES {
			// O -n apaga as constraints dos metadados do banco restaurado, entao
			// o DDL precisa ser extraido antes. A falha aqui so vira aviso: o
			// restore sem constraints continua melhor que nenhum restore.
			recoveryDir := filepath.Dir(testPath)
			constraints := preserveConstraintsBeforeNoValidity(ctx, c, report, recoveryDir)
			_ = os.Remove(testPath)
			appendLog("Tentando restore sem constraints de validade (-n)...")
			nvOutput, nvErr := runRecoveryCapture(ctx, "Restore sem constraints de validade (-n)...", gbak, restoreTestArgs(version, resolvedPort, c.User, c.Password, backup, testPath, false, true)...)
			if nvErr == nil && fileExists(testPath) {
				report.NoValidityRestore = true
				report.PartialRecovery = true
				report.action("Restore de teste concluído com -n (-no_validity): banco restaurado SEM FOREIGN KEY e SEM constraints CHECK")
				report.warning("O banco restaurado não possui as constraints originais; recrie-as manualmente após remover ou corrigir as linhas órfãs")
				writeOrphanRowsInspectionScript(report, recoveryDir, constraints)
				result := restoreTestResult{Verified: true, NoValidity: true, Path: testPath, Backup: backup}
				gstat := firebirdExe(root, version, "gstat.exe")
				if fileExists(gstat) {
					headerOutput, headerErr := runCommandCapture(ctx, gstat, "-h", testPath)
					if headerErr == nil {
						result.Header = parseDatabaseHeader(headerOutput)
					}
				}
				report.TestRestore = testPath
				return result, nil
			}
			if nvOutput != "" && !detailedLog {
				appendLog("O restore com -n também falhou. Últimas mensagens:")
				appendLog(tailText(nvOutput, 12))
			}
			report.warning("O restore com -n não concluiu; a corrupção vai além da integridade referencial")
		}
	}

	_ = os.Remove(testPath)
	if messageBox("Restore de teste falhou", "O backup foi criado, mas a restauração de teste falhou.\n\nDeseja tentar o modo de recuperação parcial (-one_at_a_time)?\n\nEsse modo restaura cada tabela separadamente e pode deixar um banco incompleto para resgate manual.", MB_YESNO|MB_ICONWARNING) != IDYES {
		return restoreTestResult{}, fmt.Errorf("backup criado, mas não passou no restore de teste")
	}

	appendLog("Tentando restore parcial, uma tabela por vez...")
	partialOutput, partialErr := runRecoveryCapture(ctx, "Restore parcial (-one_at_a_time)...", gbak, restoreTestArgs(version, resolvedPort, c.User, c.Password, backup, testPath, true, false)...)
	if partialErr == nil && fileExists(testPath) {
		report.action("Restore de teste com -one_at_a_time concluiu sem erro")
		result := restoreTestResult{Verified: true, Path: testPath, Backup: backup}
		gstat := firebirdExe(root, version, "gstat.exe")
		if fileExists(gstat) {
			headerOutput, headerErr := runCommandCapture(ctx, gstat, "-h", testPath)
			if headerErr == nil {
				result.Header = parseDatabaseHeader(headerOutput)
			}
		}
		report.TestRestore = testPath
		report.action("Restore de teste com -one_at_a_time preservado temporariamente para comparação pós-recuperação")
		return result, nil
	}
	if fileExists(testPath) {
		report.warning("Restore parcial gerou um banco incompleto para inspeção manual: " + testPath)
		report.TestRestore = testPath
		if partialOutput != "" && !detailedLog {
			appendLog(tailText(partialOutput, 12))
		}
		return restoreTestResult{Partial: true, Path: testPath, Backup: backup}, fmt.Errorf("o backup só pôde ser restaurado parcialmente; banco parcial preservado em %s", testPath)
	}
	if partialErr != nil {
		return restoreTestResult{}, fmt.Errorf("o backup não pôde ser restaurado nem em modo parcial: %w", partialErr)
	}
	return restoreTestResult{}, errors.New("o backup não pôde ser restaurado nem em modo parcial")
}

func runAdvancedRecovery(ctx context.Context, c workflowConfig) (retErr error) {
	dir, workingCopy, backup, testPath, reportPath, err := recoveryWorkspace(c)
	if err != nil {
		return fmt.Errorf("não foi possível criar a pasta de recuperação: %w", err)
	}
	technicalLogPath := filepath.Join(dir, "LOG_TECNICO_RECUPERACAO.txt")
	healthReportPath := filepath.Join(dir, "RELATORIO_SAUDE_PERFORMANCE.txt")
	setRecoveryTechnicalLog(technicalLogPath)
	logSnapshot := snapshotFirebirdLog(c.FbSource)
	report := recoveryReport{
		Started:       time.Now(),
		Source:        c.Source,
		SourceVersion: c.SourceVersion,
		TargetVersion: c.TargetVersion,
		Destination:   c.Dest,
		ReportPath:    reportPath,
		TechnicalLog:  technicalLogPath,
		SkipData: func() string {
			if versionNumber(c.SourceVersion) >= 3.0 {
				return normalizeSkipData(c.SkipData)
			}
			return ""
		}(),
	}
	report.PartialRecovery = report.SkipData != ""
	setLastReport(reportPath)
	if report.SkipData != "" {
		report.warning("O backup de recuperação excluirá dados das tabelas que corresponderem à regex -skip_data: " + report.SkipData)
	}
	defer func() {
		report.Finished = time.Now()
		if delta := collectFirebirdLogDelta(logSnapshot); delta != "" {
			report.FirebirdLogExcerpt = delta
			report.action("Trecho do firebird.log gerado durante a recuperação foi incorporado ao relatório")
		}
		if retErr != nil {
			report.fail(retErr.Error())
			if report.FinalState == "" {
				report.FinalState = "FALHA / RECUPERAÇÃO NÃO CONCLUÍDA"
			}
		}
		setLastRecoveryState(report.FinalState)
		if writeErr := writeRecoveryReport(report); writeErr != nil {
			appendLog("ATENÇÃO: não foi possível gravar o relatório de recuperação: " + writeErr.Error())
		} else {
			appendLog("Relatório de recuperação: " + reportPath)
		}
		setRecoveryTechnicalLog("")
	}()

	appendLog("Modo avançado: o banco original não será alterado por mend/reparo pesado.")
	appendLog("Pasta de recuperação: " + dir)
	appendLog("Log técnico completo: " + technicalLogPath)
	setProgress(8, "Preparando recuperação...")
	resolvedPort, endpointErr := verifyFirebirdEndpoint(ctx, "da rotina", c.SourceVersion, c.FbSource, c.Service, c.SourcePort)
	if endpointErr != nil {
		return endpointErr
	}
	originalConn := databaseConnection(resolvedPort, c.Source)
	gfix := firebirdExe(c.FbSource, c.SourceVersion, "gfix.exe")
	gbak := firebirdExe(c.FbSource, c.SourceVersion, "gbak.exe")

	setStatus("Executando diagnóstico profundo sem alterar o banco...")
	setProgress(15, "Diagnóstico somente leitura...")
	diagnosticOutput, diagnosticErr := runRecoveryCapture(ctx, "Etapa 1: Diagnóstico profundo (somente leitura)...", gfix, "-user", c.User, "-pass", c.Password, "-v", "-full", "-no_update", originalConn)
	stats := parseRecoveryStats(diagnosticOutput)
	readOnlyOnOriginal := !outputReportsUnknownSwitch(diagnosticOutput)
	if !readOnlyOnOriginal {
		report.warning("A versão do gfix não reconheceu -no_update; o Toolbox não executou validação potencialmente modificadora no original e fará o diagnóstico completo somente na cópia de trabalho")
		appendLog("ATENÇÃO: este gfix não reconhece -no_update. O diagnóstico completo será feito somente na cópia de trabalho.")
		stats = recoveryStats{}
	} else {
		if diagnosticErr != nil && !stats.HasCorruption() {
			return toolFailure("o diagnóstico somente leitura não pôde ser concluído", diagnosticErr, diagnosticOutput)
		}
		if toolOutputHasOperationalFailure(diagnosticOutput) && !stats.HasCorruption() {
			return toolFailure("o gfix não conseguiu concluir o diagnóstico", nil, diagnosticOutput)
		}
		appendLog("Diagnóstico: " + stats.Summary() + ".")
		report.action("Diagnóstico gfix -validate -full -no_update executado no banco original")
	}
	report.Diagnostic = stats
	// Só é diagnóstico de verdade quando o gfix rodou no original. Na variante
	// legada, sem -no_update, o estado ainda não foi avaliado.
	report.DiagnosticDone = readOnlyOnOriginal

	setProgress(22, "Verificando transações em limbo...")
	limboOutput, limboErr := runRecoveryCapture(ctx, "Etapa 2: Verificando transações em limbo...", gfix, "-user", c.User, "-pass", c.Password, "-list", originalConn)
	if limboErr != nil && toolOutputHasOperationalFailure(limboOutput) {
		report.warning("Não foi possível concluir a verificação de transações em limbo")
	} else {
		report.LimboCount = countLimboTransactions(limboOutput)
		if report.LimboCount > 0 {
			appendLog(fmt.Sprintf("ATENÇÃO: %d transação(ões) em limbo detectada(s). Nenhuma foi resolvida automaticamente.", report.LimboCount))
			report.warning(fmt.Sprintf("%d transação(ões) em limbo detectada(s); decisão de commit/rollback deve ser manual", report.LimboCount))
		} else {
			appendLog("✓ Nenhuma transação em limbo detectada.")
		}
	}

	setStatus("Criando cópia de trabalho protegida...")
	setProgress(30, "Criando cópia de trabalho...")
	beforeCopy, statErr := os.Stat(c.Source)
	if statErr != nil {
		return fmt.Errorf("não foi possível conferir o arquivo original antes da cópia: %w", statErr)
	}
	hash, size, copyErr := fileCopyWithSHA256(c.Source, workingCopy)
	if copyErr != nil {
		return fmt.Errorf("não foi possível criar a cópia de trabalho: %w", copyErr)
	}
	afterCopy, statErr := os.Stat(c.Source)
	if statErr != nil {
		return fmt.Errorf("a cópia foi criada, mas não foi possível reconferir o arquivo original: %w", statErr)
	}
	if beforeCopy.Size() != afterCopy.Size() || !beforeCopy.ModTime().Equal(afterCopy.ModTime()) || size != beforeCopy.Size() {
		report.warning("O arquivo original mudou enquanto a cópia de trabalho era criada")
		return errors.New("o banco original mudou durante a cópia; feche todas as aplicações e repita para evitar uma cópia inconsistente")
	}
	stableHash, stableSize, hashErr := fileSHA256(c.Source)
	if hashErr != nil {
		return fmt.Errorf("a cópia foi criada, mas não foi possível reconferir o SHA-256 do original: %w", hashErr)
	}
	if stableSize != size || !strings.EqualFold(stableHash, hash) {
		report.warning("O conteúdo do arquivo original mudou durante a cópia, embora tamanho/data possam não ter denunciado a alteração")
		return errors.New("o conteúdo do banco original mudou durante a cópia; feche todas as aplicações e repita antes de qualquer recuperação")
	}
	report.WorkingCopy = workingCopy
	report.OriginalSHA256 = stableHash
	report.OriginalSize = size
	report.action("Cópia de trabalho criada antes de qualquer reparo destrutivo")
	appendLog("✓ Cópia de trabalho criada: " + workingCopy)
	appendLog("SHA-256 estável do original: " + stableHash)
	workingConn := databaseConnection(resolvedPort, workingCopy)
	copyCheckArgs := []string{"-user", c.User, "-pass", c.Password, "-v", "-full"}
	copyCheckLabel := "Conferindo integridade e acesso à cópia de trabalho..."
	if readOnlyOnOriginal {
		copyCheckArgs = append(copyCheckArgs, "-no_update")
	} else {
		copyCheckLabel = "Executando diagnóstico de compatibilidade SOMENTE na cópia de trabalho..."
	}
	copyCheckArgs = append(copyCheckArgs, workingConn)
	copyCheckOutput, copyCheckErr := runRecoveryCapture(ctx, copyCheckLabel, gfix, copyCheckArgs...)
	copyStats := parseRecoveryStats(copyCheckOutput)
	if copyCheckErr != nil && !copyStats.HasCorruption() {
		return toolFailure("a cópia de trabalho foi criada, mas o Firebird não conseguiu abri-la com segurança", copyCheckErr, copyCheckOutput)
	}
	if toolOutputHasOperationalFailure(copyCheckOutput) && !copyStats.HasCorruption() {
		return errors.New("a cópia de trabalho foi criada, mas não ficou acessível ao serviço Firebird; escolha uma pasta de backup acessível ao serviço")
	}
	stats = mergeRecoveryStats(stats, copyStats)
	report.Diagnostic = stats
	report.PreMendDiagnostic = stats
	report.DiagnosticDone = true
	if readOnlyOnOriginal {
		report.action("Cópia de trabalho conferida em modo somente leitura antes do reparo")
	} else {
		report.action("Diagnóstico estrutural executado somente na cópia porque o gfix legado não reconheceu -no_update")
		appendLog("Diagnóstico na cópia: " + stats.Summary() + ".")
	}

	needsRecoveryFlags := stats.HasCorruption() && !stats.IndexOnly()
	if stats.IndexOnly() {
		appendLog("Corrupção restrita a índices: o Toolbox evitará mend; backup/restore tentará reconstruí-los.")
		report.action("Mend não executado porque os erros reportados estão restritos a índices")
	} else if stats.HasCorruption() {
		msg := "O diagnóstico encontrou corrupção que pode envolver perda de dados.\n\n" + stats.Summary() + "\n\nO Toolbox já criou uma cópia de trabalho e NÃO executará mend no arquivo original.\n\nDeseja tentar recuperação pesada na cópia?\n\nO gfix -mend pode descartar registros ou páginas que não consegue recuperar."
		if messageBox("Recuperação pesada", msg, MB_YESNO|MB_ICONWARNING) != IDYES {
			report.FinalState = "DIAGNÓSTICO CONCLUÍDO; REPARO PESADO NÃO AUTORIZADO"
			return errors.New("recuperação pesada cancelada; banco original preservado e relatório gerado")
		}
		setStatus("Preparando cópia corrompida para resgate...")
		setProgress(40, "Preparando cópia para resgate...")
		validationOut, validationErr := runRecoveryCapture(ctx, "Etapa 3: Validação de recuperação ignorando checksums...", gfix, "-user", c.User, "-pass", c.Password, "-v", "-full", "-ignore", workingConn)
		if validationErr != nil && !parseRecoveryStats(validationOut).HasCorruption() {
			report.warning("gfix -validate -full -ignore retornou erro antes do mend")
		}
		mendOutput, mendErr := runRecoveryCapture(ctx, "Etapa 4: Executando gfix -mend -ignore SOMENTE na cópia...", gfix, gfixMendArgs(c.User, c.Password, workingConn)...)
		mendStats := parseRecoveryStats(mendOutput)
		if toolOutputHasOperationalFailure(mendOutput) || (mendErr != nil && !mendStats.HasCorruption()) {
			if mendErr != nil {
				return fmt.Errorf("gfix -mend não conseguiu preparar a cópia para backup: %w", mendErr)
			}
			return errors.New("gfix -mend não conseguiu preparar a cópia para backup; consulte o log técnico")
		}
		report.action("gfix -mend -ignore executado somente na cópia de trabalho")
		if mendStats.HasCorruption() {
			report.warning("O mend reportou perdas/erros: " + mendStats.Summary())
			appendLog("ATENÇÃO durante mend: " + mendStats.Summary() + ".")
		}

		postArgs := []string{"-user", c.User, "-pass", c.Password, "-v", "-full", "-ignore"}
		if readOnlyOnOriginal {
			postArgs = append(postArgs, "-no_update")
		}
		postArgs = append(postArgs, workingConn)
		postOutput, postErr := runRecoveryCapture(ctx, "Etapa 4b: Revalidando a cópia após o mend...", gfix, postArgs...)
		postStats := parseRecoveryStats(postOutput)
		report.PostMendDiagnostic = postStats
		report.PostMendChecked = true
		report.action("Cópia revalidada após o mend para comparar corrupção antes/depois")
		appendLog("Pós-mend: " + postStats.Summary() + ".")
		if postErr != nil && !postStats.HasCorruption() {
			report.warning("A revalidação pós-mend terminou com erro operacional; consulte o log técnico")
		}
	} else {
		appendLog("✓ Diagnóstico não reportou corrupção estrutural; o fluxo avançado seguirá com backup e restore de teste.")
	}

	setStatus("Gerando backup de recuperação...")
	setProgress(55, "Gerando backup de recuperação...")
	usedLimbo := false
	backupArgs := advancedBackupArgs(c, workingConn, backup, needsRecoveryFlags, false)
	backupOutput, backupErr := runRecoveryCapture(ctx, "Etapa 5: Gerando backup de recuperação...", gbak, backupArgs...)
	if backupErr != nil && report.LimboCount > 0 {
		if messageBox("Backup bloqueado", "O backup falhou e existem transações em limbo.\n\nDeseja repetir o backup usando -limbo para ignorar essas transações?\n\nAs transações em limbo não serão resolvidas; apenas serão ignoradas neste backup de resgate.", MB_YESNO|MB_ICONWARNING) == IDYES {
			_ = os.Remove(backup)
			usedLimbo = true
			backupOutput, backupErr = runRecoveryCapture(ctx, "Repetindo backup com -limbo...", gbak, advancedBackupArgs(c, workingConn, backup, needsRecoveryFlags, true)...)
			report.action("Backup repetido com -limbo por decisão do usuário")
		}
	}

	if backupErr != nil || !fileExists(backup) {
		report.BackupFailureDetails = relevantGbakFailureLines(backupOutput, 48)
		// Dois sinais, duas perguntas diferentes. O log do gbak responde "qual
		// relação bloqueia o backup" e por isso é ele que escolhe o alvo do
		// -skip_data. O firebird.log responde "onde o motor quebrou" e entra no
		// relatório como diagnóstico, sem comandar nenhuma decisão.
		report.SuspectedRelation = suspectedRelationFromGbak(backupOutput)
		if report.SuspectedRelation != "" {
			report.SuspectedRelationSource = "log do gbak"
			appendLog("Possível relação afetada pelo erro do gbak: " + report.SuspectedRelation)
		}
		report.EngineCrashRelation = suspectedRelationFromFirebirdLog(collectFirebirdLogDelta(logSnapshot))
		if report.EngineCrashRelation != "" {
			appendLog("O motor registrou bugcheck ao varrer a tabela " + report.EngineCrashRelation + " (firebird.log).")
			if report.SuspectedRelation != "" && !strings.EqualFold(report.EngineCrashRelation, report.SuspectedRelation) {
				appendLog("Os dois sinais apontam tabelas diferentes: o dano não está em um único ponto.")
			}
		}
		if report.BackupFailureDetails != "" && !detailedLog {
			appendLog("Falha do backup - mensagens relevantes:")
			appendLog(report.BackupFailureDetails)
		}

		metaBackup := metadataBackupPath(backup)
		_ = os.Remove(metaBackup)
		metaOutput, metaErr := runRecoveryCapture(ctx, "Tentando preservar somente os metadados do banco...", gbak, metadataBackupArgs(c, workingConn, metaBackup)...)
		if metaErr == nil && fileExists(metaBackup) {
			report.MetadataBackup = metaBackup
			report.action("Backup somente de metadados criado após a falha do backup completo")
			appendLog("✓ Metadados preservados: " + metaBackup)
		} else {
			report.warning("Não foi possível criar o backup somente de metadados")
			if details := relevantGbakFailureLines(metaOutput, 16); details != "" {
				report.warning("Falha do backup de metadados: " + strings.ReplaceAll(details, "\n", " | "))
			}
		}

		// Antes de excluir dados de uma relação inteira, tenta a estratégia menos
		// invasiva para o caso clássico de corrupção de página de índice: desativa
		// apenas índices de usuário que não sustentam constraints, sempre na cópia
		// de trabalho, e repete o gbak. Se funcionar, nenhum registro é descartado.
		if backupErr != nil && normalizeSkipData(c.SkipData) == "" && safeRelationForSkip(report.SuspectedRelation) && gbakLooksLikeIndexCorruption(backupOutput) {
			report.IndexRescueAttempted = true
			meta, metaErr := loadRelationMetadata(ctx, c, resolvedPort, workingCopy, report.SuspectedRelation)
			if metaErr == nil {
				disabled := deactivateRelationIndexes(ctx, c, resolvedPort, workingCopy, meta)
				report.IndexesDisabled = append(report.IndexesDisabled, disabled...)
				if len(disabled) > 0 {
					appendLog("Tentando novo backup após desativar índices não vinculados a constraints em " + report.SuspectedRelation + "...")
					_ = os.Remove(backup)
					indexOutput, indexErr := runRecoveryCapture(ctx, "Repetindo backup após isolar índices da relação...", gbak, advancedBackupArgs(c, workingConn, backup, needsRecoveryFlags, usedLimbo)...)
					if indexErr == nil && fileExists(backup) {
						backupErr = nil
						backupOutput = indexOutput
						report.action("Backup completo obtido após desativar índices não vinculados a constraints somente na cópia de trabalho")
						report.warning("O backup foi obtido com índices de usuário temporariamente inativos na cópia; o restore de teste confirmará a reconstrução antes do uso")
						appendLog("✓ Backup completo obtido sem excluir os dados de " + report.SuspectedRelation + ".")
					} else {
						report.warning("A tentativa de preservar todos os registros desativando índices não vinculados a constraints não resolveu a falha")
						if details := relevantGbakFailureLines(indexOutput, 24); details != "" {
							report.BackupFailureDetails += "\n\nTentativa após desativar índices:\n" + details
						}
					}
				} else {
					report.warning("A falha parece envolver índice, mas não havia índice de usuário não vinculado a constraint que pudesse ser desativado com segurança")
				}
			} else {
				report.warning("Não foi possível ler os metadados da relação para a tentativa de resgate por índices: " + metaErr.Error())
			}
		}

		if backupErr != nil && versionNumber(c.SourceVersion) >= 3.0 && normalizeSkipData(c.SkipData) == "" && safeRelationForSkip(report.SuspectedRelation) {
			skipRegex := sqlRegexLiteral(report.SuspectedRelation)
			msg := "O gbak continua falhando ao processar a relação:\n\n" + report.SuspectedRelation + "\n\nO Toolbox pode criar um backup-base sem os dados dessa relação e, em seguida, tentar copiar de volta os registros legíveis para um banco temporário limpo.\n\nPrimeiro será usado -skip_data apenas como etapa intermediária. Depois o Toolbox tentará recuperar os registros da tabela por leitura NATURAL e, se necessário, por faixas da chave primária.\n\nDeseja iniciar esse resgate?"
			if messageBox("Tentativa de recuperação parcial", msg, MB_YESNO|MB_ICONWARNING) == IDYES {
				partialConfig := c
				partialConfig.SkipData = skipRegex
				_ = os.Remove(backup)
				appendLog("Criando backup-base sem os dados de " + report.SuspectedRelation + " para tentar o resgate dos registros...")
				partialOutput, partialErr := runRecoveryCapture(ctx, "Repetindo backup com -skip_data "+skipRegex+"...", gbak, advancedBackupArgs(partialConfig, workingConn, backup, needsRecoveryFlags, usedLimbo)...)
				if partialErr == nil && fileExists(backup) {
					backupErr = nil
					backupOutput = partialOutput
					report.SkipData = skipRegex
					report.PartialRecovery = true
					baseBackup := basePartialBackupPath(backup, report.SuspectedRelation)
					_ = os.Remove(baseBackup)
					if renameErr := os.Rename(backup, baseBackup); renameErr != nil {
						baseBackup = backup
						report.warning("Não foi possível renomear o backup-base para identificação forense: " + renameErr.Error())
					}
					report.BasePartialBackup = baseBackup
					report.warning("Backup-base criado com -skip_data para permitir tentativa de resgate dos registros de " + report.SuspectedRelation)
					report.action("Backup-base criado com -skip_data preservando as demais tabelas")
					appendLog("✓ Backup-base criado. Tentando recuperar os registros de " + report.SuspectedRelation + "...")

					rowResult, rowErr := attemptRowRescue(ctx, c, resolvedPort, workingCopy, baseBackup, backup, report.SuspectedRelation, report.IndexesDisabled)
					report.RowRescueAttempted = rowResult.Attempted
					report.RowRescueRelation = report.SuspectedRelation
					report.RowRescueRecovered = rowResult.Recovered
					report.RowRescueLostKeys = append(report.RowRescueLostKeys, rowResult.LostKeys...)
					report.RowRescueComplete = rowResult.Complete
					report.RowRescueStage = rowResult.StagePath
					if rowErr == nil && rowResult.Backup != "" && fileExists(rowResult.Backup) {
						backup = rowResult.Backup
						backupErr = nil
						report.action(fmt.Sprintf("Resgate de registros de %s concluído: %d registro(s) copiado(s)", report.SuspectedRelation, rowResult.Recovered))
						if rowResult.Complete {
							report.PartialRecovery = false
							report.SkipData = ""
							report.warning("O -skip_data foi usado apenas no backup-base intermediário; os registros da relação foram reinseridos no banco limpo antes do backup final")
							appendLog(fmt.Sprintf("✓ %d registro(s) de %s recuperados. Nenhuma chave foi perdida.", rowResult.Recovered, report.SuspectedRelation))
						} else {
							report.PartialRecovery = true
							report.SkipData = ""
							report.warning(fmt.Sprintf("Resgate parcial de %s: %d registro(s) recuperado(s), %d chave(s) não recuperada(s)", report.SuspectedRelation, rowResult.Recovered, len(rowResult.LostKeys)))
							appendLog(fmt.Sprintf("ATENÇÃO: %d registro(s) recuperado(s) de %s; %d chave(s) não puderam ser copiadas.", rowResult.Recovered, report.SuspectedRelation, len(rowResult.LostKeys)))
							if messageBox("Resgate parcial de registros", fmt.Sprintf("Foram recuperados %d registros de %s, mas %d chave(s) não puderam ser copiadas.\n\nDeseja continuar com esta recuperação parcial?", rowResult.Recovered, report.SuspectedRelation, len(rowResult.LostKeys)), MB_YESNO|MB_ICONWARNING) != IDYES {
								backupErr = errors.New("recuperação parcial de registros não autorizada pelo usuário")
							}
						}
					} else {
						if rowErr == nil {
							rowErr = errors.New("o resgate terminou sem gerar o backup pós-resgate esperado")
						}
						report.warning("O backup-base foi criado, mas o resgate automático dos registros falhou: " + rowErr.Error())
						appendLog("ATENÇÃO: não foi possível recuperar os registros automaticamente: " + rowErr.Error())
						if messageBox("Resgate dos registros falhou", "O restante do banco foi preservado, mas o resgate automático dos registros de "+report.SuspectedRelation+" falhou.\n\nDeseja continuar usando o backup-base SEM os dados dessa relação?", MB_YESNO|MB_ICONWARNING) != IDYES {
							backupErr = rowErr
						} else {
							backup = baseBackup
							report.warning("Usuário optou por continuar com o backup-base sem os dados da relação após falha do resgate")
							report.PartialRecovery = true
						}
					}
				} else {
					report.warning("A tentativa automática com -skip_data também falhou")
					if details := relevantGbakFailureLines(partialOutput, 32); details != "" {
						report.BackupFailureDetails += "\n\nTentativa com -skip_data:\n" + details
					}
					backupErr = partialErr
				}
			} else {
				report.warning("Tentativa de backup-base e resgate de registros não autorizada pelo usuário; nenhum dado foi excluído automaticamente")
			}
		}
	}

	if backupErr != nil || !fileExists(backup) {
		if versionNumber(c.SourceVersion) >= 3.0 && normalizeSkipData(c.SkipData) == "" && report.SuspectedRelation == "" {
			report.warning("O backup falhou e não foi possível identificar automaticamente uma relação específica; use o log técnico para localizar a tabela e, se apropriado, informe-a em 'Pular tabelas (regex)'.")
		}
		report.FinalState = "NÃO RECUPERÁVEL COM FERRAMENTAS PADRÃO"
		if report.MetadataBackup != "" {
			report.FinalState += "; METADADOS PRESERVADOS"
		}
		if report.SuspectedRelation != "" {
			if report.EngineCrashRelation != "" && !strings.EqualFold(report.EngineCrashRelation, report.SuspectedRelation) {
				return fmt.Errorf("backup de recuperação falhou; possível relação afetada: %s, e o motor sofreu bugcheck ao varrer %s. Metadados e logs foram preservados para a próxima tentativa", report.SuspectedRelation, report.EngineCrashRelation)
			}
			return fmt.Errorf("backup de recuperação falhou; possível relação afetada: %s. Metadados e logs foram preservados para a próxima tentativa", report.SuspectedRelation)
		}
		return errors.New("nem gfix/gbak conseguiram produzir um backup de dados utilizável; preserve o original, a cópia, os metadados e o log técnico para recuperação de baixo nível")
	}

	report.action("Backup de recuperação criado")
	report.Backup = backup
	setLastBackup(backup)
	appendLog("✓ Backup de recuperação criado: " + backup)

	setStatus("Validando o backup com restore de teste...")
	testResult, testErr := verifyBackupByRestore(ctx, c, backup, testPath, &report)
	if testErr != nil {
		if testResult.Partial {
			report.TestRestoreState = "PARCIAL - banco preservado para inspeção"
			if report.FinalState == "" {
				report.FinalState = "RECUPERAÇÃO PARCIAL DISPONÍVEL PARA INSPEÇÃO; RESTORE NÃO COMPLETO"
			}
		} else {
			report.TestRestoreState = "FALHOU"
			if report.FinalState == "" {
				report.FinalState = "BACKUP CRIADO MAS NÃO VALIDADO"
			}
		}
		return testErr
	}
	if testResult.Backup != "" && !strings.EqualFold(filepath.Clean(testResult.Backup), filepath.Clean(backup)) {
		backup = testResult.Backup
		report.Backup = backup
		setLastBackup(backup)
		report.action("Backup saneado substituiu o backup intermediário após novo restore de teste aprovado")
	}
	report.TestRestoreState = "VERIFICADO COM SUCESSO"
	report.TestODS = testResult.Header.ODS
	report.TestDialect = testResult.Header.Dialect
	report.TestPageSize = testResult.Header.PageSize
	if testResult.Header.ODS != "" || testResult.Header.Dialect != "" || testResult.Header.PageSize != "" {
		setLastHeader(testResult.Header)
	}
	if testResult.Header.ODS != "" {
		appendLog("✓ Backup verificado por restore real | ODS " + testResult.Header.ODS + " | Dialeto " + testResult.Header.Dialect + ".")
	} else {
		appendLog("✓ Backup verificado por uma restauração real.")
	}
	setProgress(86, "Backup verificado")

	if c.TargetVersion != "" {
		if fileExists(c.Dest) {
			if messageBox("Destino existente", "O arquivo de destino já existe.\n\nO backup já foi verificado por restore de teste. Deseja continuar e substituir/restaurar o destino?", MB_YESNO|MB_ICONWARNING) != IDYES {
				report.warning("Restauração final cancelada; backup verificado foi preservado")
				if report.PartialRecovery {
					report.FinalState = "RECUPERAÇÃO PARCIAL VERIFICADA; RESTAURAÇÃO FINAL CANCELADA"
				} else {
					report.FinalState = "BACKUP COMPLETO RECUPERADO E VERIFICADO; RESTAURAÇÃO FINAL CANCELADA"
				}
				return nil
			}
		}
		setProgress(90, "Restaurando destino final...")
		if err := restoreBackup(ctx, c, backup, report.NoValidityRestore); err != nil {
			if report.PartialRecovery {
				report.FinalState = "BACKUP PARCIAL VERIFICADO; FALHA NA RESTAURAÇÃO FINAL"
			} else {
				report.FinalState = "BACKUP COMPLETO VERIFICADO; FALHA NA RESTAURAÇÃO FINAL"
			}
			return err
		}
		if len(report.IndexesDisabled) > 0 && !report.DuplicateRepairApplied {
			version, root, port := testRestoreEngine(c)
			resolved := port
			if supportsServiceManagerRestore(version) {
				if p, resolveErr := verifyFirebirdEndpoint(ctx, "da reativação final de índices", version, root, "", port); resolveErr == nil {
					resolved = p
				}
			}
			if _, activeErr := runISQLScript(ctx, root, version, resolved, c.User, c.Password, c.Dest, "Reativando índices no banco final...", buildAlterIndexesScript(report.IndexesDisabled, true)); activeErr != nil {
				report.FinalState = "RESTAURAÇÃO FINAL CRIADA, MAS ÍNDICES NÃO FORAM REATIVADOS"
				return fmt.Errorf("o banco final foi restaurado, mas não foi possível reativar os índices: %w", activeErr)
			}
			report.action("Índices reconstruídos/reativados no banco final")
		}
		report.action("Restauração final concluída no Firebird " + c.TargetVersion)
		if report.PartialRecovery {
			if report.DuplicateRepairApplied && report.DuplicateRowsRemoved > 0 {
				report.FinalState = fmt.Sprintf("RECUPERAÇÃO PARCIAL CONCLUÍDA; %d DUPLICIDADE(S) DE CHAVE RESOLVIDA(S) COM DECISÃO EXPLÍCITA", report.DuplicateRowsRemoved)
			} else if report.RowRescueAttempted && len(report.RowRescueLostKeys) > 0 {
				report.FinalState = fmt.Sprintf("RECUPERAÇÃO PARCIAL CONCLUÍDA; %d CHAVE(S) NÃO RECUPERADA(S) EM %s", len(report.RowRescueLostKeys), report.RowRescueRelation)
			} else if report.SkipData != "" {
				report.FinalState = "RECUPERAÇÃO PARCIAL CONCLUÍDA; DADOS EXCLUÍDOS: " + report.SkipData
			} else {
				report.FinalState = "RECUPERAÇÃO PARCIAL CONCLUÍDA"
			}
		} else if report.DuplicateRepairApplied && report.DuplicateRowsRemoved > 0 {
			report.FinalState = fmt.Sprintf("RECUPERAÇÃO COMPLETA CONCLUÍDA; %d DUPLICIDADE(S) ESTRUTURAL(IS) DE CHAVE REMOVIDA(S)", report.DuplicateRowsRemoved)
		} else if report.RowRescueAttempted && report.RowRescueComplete {
			report.FinalState = fmt.Sprintf("RECUPERAÇÃO COMPLETA CONCLUÍDA; %d REGISTRO(S) DE %s RESGATADO(S)", report.RowRescueRecovered, report.RowRescueRelation)
		} else {
			report.FinalState = "RECUPERAÇÃO COMPLETA CONCLUÍDA"
		}
	} else {
		if report.PartialRecovery {
			if report.DuplicateRepairApplied && report.DuplicateRowsRemoved > 0 {
				report.FinalState = fmt.Sprintf("BACKUP PARCIAL CRIADO E VERIFICADO; %d DUPLICIDADE(S) DE CHAVE RESOLVIDA(S) COM DECISÃO EXPLÍCITA", report.DuplicateRowsRemoved)
			} else if report.RowRescueAttempted && len(report.RowRescueLostKeys) > 0 {
				report.FinalState = fmt.Sprintf("BACKUP PARCIAL CRIADO E VERIFICADO; %d CHAVE(S) NÃO RECUPERADA(S) EM %s", len(report.RowRescueLostKeys), report.RowRescueRelation)
			} else if report.SkipData != "" {
				report.FinalState = "BACKUP PARCIAL DE RECUPERAÇÃO CRIADO E VERIFICADO; DADOS EXCLUÍDOS: " + report.SkipData
			} else {
				report.FinalState = "BACKUP PARCIAL DE RECUPERAÇÃO CRIADO E VERIFICADO"
			}
		} else if report.DuplicateRepairApplied && report.DuplicateRowsRemoved > 0 {
			report.FinalState = fmt.Sprintf("BACKUP COMPLETO CRIADO E VERIFICADO; %d DUPLICIDADE(S) ESTRUTURAL(IS) DE CHAVE REMOVIDA(S)", report.DuplicateRowsRemoved)
		} else if report.RowRescueAttempted && report.RowRescueComplete {
			report.FinalState = fmt.Sprintf("BACKUP COMPLETO CRIADO E VERIFICADO; %d REGISTRO(S) DE %s RESGATADO(S)", report.RowRescueRecovered, report.RowRescueRelation)
		} else {
			report.FinalState = "BACKUP COMPLETO DE RECUPERAÇÃO CRIADO E VERIFICADO"
		}
	}

	healthDB := testPath
	healthReference := ""
	if c.TargetVersion != "" {
		healthDB = c.Dest
		healthReference = testPath
	}
	report.HealthReport = healthReportPath
	health, healthErr := runPostRecoveryHealth(ctx, c, healthDB, healthReference, healthReportPath, &report)
	report.Health = health
	report.HealthChecked = true
	if healthErr != nil {
		if !strings.Contains(strings.ToUpper(report.FinalState), "NÃO APTO PARA PRODUÇÃO") {
			if report.FinalState != "" {
				report.FinalState += "; "
			}
			report.FinalState += "NÃO APTO PARA PRODUÇÃO"
		}
		if fileExists(testPath) {
			report.TestRestore = testPath
			report.warning("Restore de referência preservado porque um gate de pós-recuperação falhou; use-o para comparar o destino sem repetir todo o resgate")
		}
		appendLog("ERRO: pós-recuperação reprovou o banco. Consulte RELATORIO_SAUDE_PERFORMANCE.txt antes de qualquer uso.")
		return healthErr
	}
	report.action("Pós-recuperação aprovado: banco acessível, sweep final concluído, validação estrutural sem corrupção e nenhum índice/PK/UNIQUE inativo")
	if c.TargetVersion != "" {
		if report.NoValidityRestore {
			report.FinalState += "; RESTAURADO SEM CONSTRAINTS (-n); RECRIAR FOREIGN KEY ANTES DE USAR"
		} else if report.PartialRecovery {
			report.FinalState += "; ESTRUTURALMENTE SAUDÁVEL; REVISÃO FUNCIONAL OBRIGATÓRIA"
		} else {
			report.FinalState += "; APTO PARA PRODUÇÃO"
		}
	} else {
		report.FinalState += "; SAÚDE DO RESTORE DE TESTE APROVADA"
	}
	setProgress(99, "Pós-recuperação aprovada")
	appendLog("✓ Pós-recuperação aprovada: " + health.Readiness + ".")
	appendLog("Relatório de saúde: " + healthReportPath)
	cleanupTestRestoreAfterHealth(testPath, &report)
	if c.TargetVersion != "" && c.Dest != "" {
		appendLog("IMPORTANTE: o Toolbox não troca automaticamente o banco usado pela aplicação. Use/configure o sistema para o arquivo final aprovado: " + c.Dest)
		report.warning("O banco original não foi substituído automaticamente; antes de reabrir o sistema, confirme que a aplicação aponta para o destino final aprovado")
	}
	return nil
}

func cleanupTestRestoreAfterHealth(testPath string, report *recoveryReport) {
	if strings.TrimSpace(testPath) == "" || !fileExists(testPath) {
		return
	}
	if err := os.Remove(testPath); err != nil {
		report.warning("Pós-recuperação aprovada, mas não foi possível excluir o restore de referência: " + err.Error())
		report.TestRestore = testPath
		return
	}
	report.TestRestore = "temporário removido após comparação pós-recuperação aprovada"
	report.action("Restore de referência comparado com o destino final e removido após aprovação dos gates")
}

func runRepair(ctx context.Context, c workflowConfig, advanced bool) error {
	if advanced {
		return runAdvancedRecovery(ctx, c)
	}
	gfix := firebirdExe(c.FbSource, c.SourceVersion, "gfix.exe")
	gbak := firebirdExe(c.FbSource, c.SourceVersion, "gbak.exe")
	backup := backupFilePath(c)

	resolvedPort, endpointErr := verifyFirebirdEndpoint(ctx, "da rotina", c.SourceVersion, c.FbSource, c.Service, c.SourcePort)
	if endpointErr != nil {
		return endpointErr
	}
	dbConn := databaseConnection(resolvedPort, c.Source)

	setStatus("Validando banco com Firebird " + c.SourceVersion + "...")
	setProgress(12, "Validando banco...")
	validationOutput, validationErr := runRecoveryCapture(ctx, "Etapa 1: Validando banco...", gfix, "-user", c.User, "-pass", c.Password, "-v", "-full", dbConn)
	validationStats := parseRecoveryStats(validationOutput)
	if validationErr != nil || validationStats.HasCorruption() || toolOutputHasOperationalFailure(validationOutput) {
		return fmt.Errorf("a validação encontrou problema; use 'Recuperar banco corrompido' para diagnóstico seguro e recuperação")
	}
	appendLog("✓ Banco validado.")
	setProgress(25, "Banco validado")

	setStatus("Executando sweep...")
	setProgress(32, "Executando sweep...")
	sweepOutput, sweepErr := runRecoveryCapture(ctx, "Etapa 2: Executando sweep...", gfix, "-user", c.User, "-pass", c.Password, "-sweep", dbConn)
	if sweepErr != nil {
		return fmt.Errorf("falha ao executar sweep: %w", sweepErr)
	}
	if toolOutputHasOperationalFailure(sweepOutput) {
		return errors.New("falha ao executar sweep; o Firebird antigo reportou erro na saída mesmo com código de retorno zero")
	}
	appendLog("✓ Sweep concluído.")
	setProgress(43, "Sweep concluído")

	setStatus("Gerando backup...")
	setProgress(50, "Gerando backup...")
	if err := runCommand(ctx, "Etapa 3: Gerando backup...", gbak, "-b", "-v", "-user", c.User, "-pass", c.Password, dbConn, backup); err != nil {
		return fmt.Errorf("falha ao gerar backup: %w", err)
	}
	if !fileExists(backup) {
		return fmt.Errorf("o gbak terminou sem criar o backup esperado: %s", backup)
	}
	appendLog("✓ Backup criado: " + backup)
	setLastBackup(backup)
	setProgress(62, "Backup criado")

	if c.TargetVersion == "" {
		setStatus("Correção concluída; backup gerado com sucesso.")
		return nil
	}

	if fileExists(c.Dest) {
		if messageBox("Destino existente", "O arquivo de destino já existe.\n\nA restauração pode substituir o arquivo. Deseja continuar?", MB_YESNO|MB_ICONWARNING) != IDYES {
			appendLog("Restauração cancelada; o backup foi preservado em: " + backup)
			return nil
		}
	}
	return restoreBackup(ctx, c, backup, false)
}

func runRestoreOnly(ctx context.Context, c workflowConfig) error {
	if c.TargetVersion == "" {
		return errors.New("selecione a versão de restauração")
	}
	if fileExists(c.Dest) {
		if messageBox("Destino existente", "O arquivo de destino já existe.\n\nDeseja continuar com a restauração?", MB_YESNO|MB_ICONWARNING) != IDYES {
			return errors.New("restauração cancelada pelo usuário")
		}
	}
	return restoreBackup(ctx, c, c.Source, false)
}

func executeWorkflow() {
	c := collectConfig()
	if err := validateConfig(c); err != nil {
		messageBox("Firebird Toolbox", err.Error()+".", MB_OK|MB_ICONWARNING)
		return
	}

	setStatus("Pré-validando arquivos, Firebird, gravação e espaço em disco...")
	setText(idResult, "Verificando pré-requisitos...")
	preflight, preflightErr := runPreflight(c)
	if preflightErr != nil {
		setStatus("Pré-validação reprovada. Corrija o item informado antes de executar.")
		setText(idResult, "⚠ Pré-validação reprovada")
		messageBox("Pré-validação", "A rotina não foi iniciada.\n\n"+preflightErr.Error(), MB_OK|MB_ICONWARNING)
		return
	}
	setText(idResult, "Aguardando execução")
	setStatus("Pré-validação concluída. Pronto para confirmar a rotina.")
	savePreferences(c)

	if c.Routine.Kind == routineRepair || c.Routine.Kind == routineRepairAdvanced {
		text := "Feche os sistemas conectados ao banco antes de continuar."
		if c.Routine.Kind == routineRepairAdvanced {
			text = "O modo avançado fará diagnóstico profundo somente leitura, criará uma cópia de trabalho com SHA-256 e executará qualquer mend/reparo pesado somente nessa cópia.\n\nSe o backup falhar em uma relação, o Toolbox tentará primeiro contornar índices corrompidos sem excluir dados. Se ainda for necessário usar -skip_data como base intermediária, tentará recolocar os registros legíveis da tabela por leitura NATURAL e, quando possível, isolar somente as chaves irrecuperáveis.\n\nTodo backup final passará por restore de teste antes de ser considerado válido.\n\n" + text
		} else {
			text = "A rotina executará validação, sweep e backup do banco.\n\n" + text
		}
		if c.TargetVersion != "" {
			text += "\n\nDepois, o backup será restaurado com Firebird " + c.TargetVersion + "."
		}
		if c.Routine.Kind == routineRepairAdvanced && versionNumber(c.SourceVersion) >= 3.0 && normalizeSkipData(c.SkipData) != "" {
			text += "\n\nATENÇÃO: o campo Pular tabelas está preenchido. Os dados das tabelas que corresponderem à regex NÃO entrarão no backup de recuperação."
		}
		if len(preflight.Warnings) > 0 {
			text += "\n\nAvisos da pré-validação:\n- " + strings.Join(preflight.Warnings, "\n- ")
		}
		if messageBox("Confirmar rotina", text+"\n\nDeseja executar agora?", MB_YESNO|MB_ICONWARNING) != IDYES {
			return
		}
	}

	if (c.Routine.Kind == routineRepair || c.Routine.Kind == routineRepairAdvanced) && isBackwardMigration(c.SourceVersion, c.TargetVersion) {
		msg := "A versão de destino é anterior à versão usada para gerar o backup.\n\n" +
			"Firebird " + c.SourceVersion + " -> Firebird " + c.TargetVersion + "\n\n" +
			"Migrações para versões anteriores não têm compatibilidade garantida e podem falhar se o banco usar recursos ou metadados mais novos.\n\nDeseja tentar mesmo assim?"
		if messageBox("Migração para versão anterior", msg, MB_YESNO|MB_ICONWARNING) != IDYES {
			return
		}
	}

	clearLog()
	resetRunResult()
	startRunTimer()
	appendLog("Iniciando rotina: " + c.Routine.Name)
	appendLog("Cronômetro iniciado: " + time.Now().Format("15:04:05"))
	appendLog("Pré-validação concluída antes da execução:")
	appendLog(preflight.text())
	if c.Routine.Kind != routineRestore {
		appendLog("Firebird da rotina: " + c.SourceVersion + " | Porta: " + c.SourcePort)
		appendLog("Instalação Firebird da rotina: " + c.FbSource)
	}
	if c.TargetVersion != "" {
		appendLog("Restaurar para: Firebird " + c.TargetVersion + " | Porta: " + c.TargetPort)
		appendLog("Instalação Firebird da restauração: " + c.FbTarget)
	}
	appendLog("Origem: " + c.Source)
	if detailedLog {
		appendLog("Log técnico: detalhado (gbak com saída verbose).")
	} else {
		appendLog("Log: resumido. Apenas etapas, avisos, erros e resultado final serão exibidos.")
	}
	if c.Dest != "" {
		appendLog("Destino: " + c.Dest)
	}
	if c.Routine.Kind == routineRepairAdvanced && versionNumber(c.SourceVersion) >= 3.0 && normalizeSkipData(c.SkipData) != "" {
		appendLog("Recuperação: -skip_data " + normalizeSkipData(c.SkipData))
	}

	ctx, cancel := context.WithCancel(context.Background())
	runMu.Lock()
	currentCancel = cancel
	runMu.Unlock()
	setRunning(true)
	setStatus("Executando " + c.Routine.Name + "...")

	go func() {
		defer func() {
			cancel()
			runMu.Lock()
			currentCancel = nil
			runMu.Unlock()
			setRunning(false)
		}()

		var err error
		switch c.Routine.Kind {
		case routineRepair:
			err = runRepair(ctx, c, false)
		case routineRepairAdvanced:
			err = runRepair(ctx, c, true)
		case routineRestore:
			err = runRestoreOnly(ctx, c)
		}

		if errors.Is(err, context.Canceled) {
			elapsed := finishRunTimer()
			appendLog("Rotina interrompida pelo usuário.")
			appendLog("Tempo total de execução: " + elapsed)
			setStatus("Rotina interrompida.")
			setText(idResult, "■ Interrompido")
			setTerminalProgress(PBST_PAUSED, "Interrompido pelo usuário")
			updateResultButtons()
			return
		}
		if err != nil {
			elapsed := finishRunTimer()
			appendLog("ERRO: " + err.Error())
			appendLog("Tempo total de execução: " + elapsed)
			state := strings.ToUpper(recoveryStateSnapshot())
			if strings.Contains(state, "NÃO APTO PARA PRODUÇÃO") {
				setStatus("Banco recuperado, mas reprovado nos gates de saúde; não usar em produção.")
				setText(idResult, "⚠ Não apto para produção")
				setTerminalProgress(PBST_ERROR, "Não apto para produção")
				messageBox("Banco não apto para produção", failureSummary(err), MB_OK|MB_ICONERROR)
			} else if strings.Contains(state, "AGUARDANDO DECISÃO") || strings.Contains(state, "PARCIAL") {
				if strings.Contains(state, "AGUARDANDO DECISÃO") {
					setStatus("Conflito de dados preservado; decisão manual necessária.")
					setText(idResult, "⚠ Conflito aguardando decisão")
					setTerminalProgress(PBST_PAUSED, "Aguardando decisão manual")
					messageBox("Recuperação aguardando decisão", failureSummary(err), MB_OK|MB_ICONWARNING)
				} else {
					setStatus("Recuperação parcial disponível; revise o relatório.")
					setText(idResult, "⚠ Recuperação parcial")
					setTerminalProgress(PBST_PAUSED, "Recuperação parcial disponível")
					messageBox("Recuperação parcial", failureSummary(err), MB_OK|MB_ICONWARNING)
				}
			} else {
				setStatus("Rotina encerrada com erro. Consulte o log.")
				setText(idResult, "⚠ Erro na rotina")
				setTerminalProgress(PBST_ERROR, "Encerrado com erro")
				messageBox("Erro na rotina", failureSummary(err), MB_OK|MB_ICONERROR)
			}
			updateResultButtons()
			return
		}
		elapsed := finishRunTimer()
		appendLog("Tempo total de execução: " + elapsed)
		state := strings.ToUpper(recoveryStateSnapshot())
		dialogText, compact := successSummary(c)
		setText(idResult, compact)
		updateResultButtons()
		if strings.Contains(state, "PARCIAL") {
			setTerminalProgress(PBST_PAUSED, "Recuperação parcial concluída")
			appendLog("⚠ Recuperação parcial concluída. Consulte o relatório antes de usar o banco.")
			setStatus("Recuperação parcial concluída; revise o relatório.")
			messageBox("⚠ Recuperação parcial", dialogText, MB_OK|MB_ICONWARNING)
		} else {
			setTerminalProgress(PBST_NORMAL, "Concluído com sucesso")
			appendLog("\u2713 Rotina concluída com sucesso.")
			setStatus("Concluído com sucesso.")
			messageBox("✓ Rotina concluída", dialogText, MB_OK|MB_ICONINFO)
		}
	}()
}

func stopWorkflow() {
	runMu.Lock()
	cancel := currentCancel
	active := running
	runMu.Unlock()
	if !active || cancel == nil {
		return
	}
	if messageBox("Interromper rotina", "Deseja interromper a rotina em execução?", MB_YESNO|MB_ICONWARNING) != IDYES {
		return
	}
	appendLog("Cancelamento solicitado pelo usuário; aguardando encerramento seguro do processo Firebird...")
	setStatus("Cancelamento solicitado — aguardando gbak/gfix encerrar com segurança...")
	setText(idProgressText, "Cancelamento solicitado • aguardando encerramento seguro...")
	setProgressState(PBST_PAUSED)
	cancel()
}

func helpText() string {
	return strings.TrimSpace(`FIREBIRD TOOLBOX - AJUDA

VISÃO GERAL
O Firebird Toolbox centraliza manutenção, correção, restauração e recuperação de bancos Firebird. O programa executa as ferramentas oficiais gfix, gbak e gstat e mantém o log dentro da própria interface.

CRONÔMETRO DE EXECUÇÃO
Ao clicar em Executar e confirmar a rotina, o campo Tempo decorrido começa em 0 min 00 s e é atualizado a cada segundo. A contagem é congelada quando o processo termina, inclusive em sucesso, recuperação parcial, conflito, erro ou cancelamento. O tempo total também é registrado no log e nas confirmações finais.

1. INÍCIO RÁPIDO
1) Escolha a rotina.
2) Informe a versão do Firebird que abrirá o banco de origem.
3) Se desejar restauração, escolha "Restaurar para".
4) Selecione o banco de origem, o destino e a pasta de backup.
5) Confira os caminhos do Firebird, as portas e as credenciais.
6) Clique em Executar. Antes de iniciar, o Toolbox confere arquivo, versão real do Firebird, porta, gravação e espaço livre.
7) Acompanhe as fases, o progresso e o log. Ao final, use Localizar banco, Abrir relatório ou Abrir pasta conforme o resultado disponível.

PRÉ-VALIDAÇÃO
Antes de iniciar uma rotina longa, o Toolbox verifica se a origem existe, confirma a versão pelo gbak quando possível, testa a porta Firebird, testa gravação nas pastas e confere espaço livre. Se um requisito crítico falhar, a rotina não começa.

PREFERÊNCIAS
Rotina, versões e caminhos podem ser lembrados entre sessões em preferences.json dentro da pasta de configuração do usuário. A senha nunca é gravada.

2. ROTINAS
Manutenção / conversão
- Valida o banco, executa sweep, gera backup e, quando selecionado, converte/restaura em outra instância/versão.
- É o modo recomendado para manutenção preventiva, backup validado e migração entre versões quando o banco não apresenta corrupção estrutural.

Recuperar banco corrompido
- Indicado para corrupção real, erros de página, BLOB, índice ou falhas de gbak.
- Começa com diagnóstico somente leitura quando a versão suporta esse recurso.
- Cria uma cópia de trabalho antes de qualquer reparo destrutivo.
- O gfix -mend, quando necessário e autorizado, é executado somente na cópia de trabalho.
- Gera backup de recuperação, executa restore real de teste e cria relatório técnico.
- Se a falha parecer estar em índice, tenta primeiro preservar todos os dados desativando apenas índices de usuário não ligados a constraints na cópia de trabalho.
- Se uma relação específica continuar impedindo o backup, usa -skip_data apenas como base intermediária e tenta recolocar os registros legíveis no banco limpo.
- A recuperação dos registros começa com leitura NATURAL, evitando índices corrompidos. Se necessário e houver chave primária numérica simples, o Toolbox divide a faixa da chave até isolar registros que realmente não podem ser lidos.
- Se o restore falhar ao ativar uma PRIMARY KEY/UNIQUE por chave duplicada, o Toolbox NÃO usa -one_at_a_time: refaz a restauração com gbak -inactive (todos os índices inativos), preserva uma cópia para auditoria, identifica a chave problemática e compara as ocorrências.
- Duplicidades logicamente idênticas podem ser saneadas automaticamente na cópia de trabalho, mantendo a primeira ocorrência física. Se houver campos diferentes, o Toolbox abre um resolvedor com REGISTRO 1 e REGISTRO 2 lado a lado, compara BLOBs/Fotos por SHA-256 quando a engine suporta, permite escolher explicitamente qual ocorrência manter, abrir a comparação exportada ou preservar ambos e cancelar. Após a escolha, a recuperação continua automaticamente.
- Depois de cada decisão, o Toolbox tenta reativar os índices individualmente. Se a ativação revelar OUTRA chave duplicada, o resolvedor abre novamente e o processo continua em loop, sem abandonar a recuperação após o primeiro conflito.
- Se não houver nova duplicidade detectável, mas algum índice/PRIMARY KEY continuar inativo, no Firebird 3.0+ o Toolbox usa um fallback de reconstrução limpa: cria uma base sem os dados daquela relação, restaura os metadados com índices inativos, copia todos os registros por leitura NATURAL para a estrutura nova, reativa triggers e índices, confere a contagem e gera um novo backup.
- Depois do saneamento/reconstrução, um novo backup é criado e o restore de teste completo é repetido.
- Depois da restauração final, executa pós-recuperação: acesso SQL, sweep final, gfix -validate -full, transações em limbo, índices/PRIMARY KEY/UNIQUE inativos, header e contagens essenciais. Um banco com gate crítico reprovado NÃO é marcado como apto para produção.

Somente restaurar
- Restaura um arquivo .BAK/.FBK para o banco de destino usando a versão selecionada.

3. VERSÕES DO FIREBIRD
São oferecidas as linhas 1.0, 1.5, 2.0, 2.1, 2.5, 3.0, 4.0 e 5.0.
- O Toolbox procura gbak.exe, gfix.exe e gstat.exe na raiz e/ou pasta bin, conforme a versão.
- Restaurar para uma versão posterior é o caminho mais comum de migração.
- Restaurar para uma versão anterior pode falhar por incompatibilidade de metadados ou recursos. O Toolbox avisa, mas não pode garantir a conversão regressiva.

4. CAMPO "PULAR TABELAS (REGEX)"
Disponível no modo avançado para Firebird 3.0 ou superior.
- Usa a opção gbak -skip_data.
- Exclui os DADOS das relações correspondentes, mas preserva a estrutura/metadados.
- Deve ser usado manualmente apenas quando você realmente deseja excluir os dados correspondentes.
- Quando o Toolbox identifica uma relação suspeita automaticamente, o -skip_data passa a ser uma etapa intermediária: depois ele tenta recuperar os registros da relação e gerar um novo backup com os dados recuperados.
- Se todos os registros forem reinseridos e o novo backup passar no restore de teste, o resultado pode voltar a ser RECUPERAÇÃO COMPLETA.
- Se algumas chaves não puderem ser recuperadas, o resultado é RECUPERAÇÃO PARCIAL e o relatório lista essas chaves.

5. RESULTADOS DA RECUPERAÇÃO
RECUPERAÇÃO COMPLETA
- Backup criado, restore de teste aprovado e nenhum dado foi excluído deliberadamente.

RECUPERAÇÃO PARCIAL
- O banco foi restaurado e validado, mas alguma relação continuou sem todos os registros originais ou uma duplicidade com conteúdo diferente/inconclusivo foi resolvida por decisão explícita do usuário.
- Quando o resgate por chave primária foi possível, o relatório informa quantos registros foram recuperados e quais chaves não puderam ser copiadas.
- Revise obrigatoriamente o relatório antes de colocar o banco em produção.

BACKUP CRIADO MAS NÃO VALIDADO
- Existe um arquivo de backup, porém o restore de teste não foi aprovado. Não trate esse backup como seguro.

NÃO RECUPERÁVEL COM FERRAMENTAS PADRÃO
- gfix/gbak não conseguiram produzir um banco validado. Preserve o original e todos os arquivos de recuperação para análise especializada.

6. ARQUIVOS GERADOS NO MODO AVANÇADO
Dentro da pasta RECOVERY_... podem ser criados:
- *_WORK.RED: cópia de trabalho usada para reparos.
- *_RECOVERY.BAK: backup de recuperação.
- *_RECOVERY_SEM_<TABELA>.BAK ou backup-base equivalente: cópia intermediária sem a relação problemática, preservada para auditoria quando usada.
- *_SALVAGE_<TABELA>.FDB: banco temporário limpo onde os registros legíveis são reinseridos.
- *_COM_ITENS.BAK: novo backup gerado depois do resgate de registros, quando aplicável.
- *_RESTORE_FALHO_XX.RED: restore originalmente falho preservado quando o erro envolve chave duplicada.
- *_CONFLITO_ORIGINAL_XX.RED: restauração completa com índices inativos, preservada antes do saneamento das duplicidades.
- *_CONFLITO_SANEADO_XX.RED: banco de trabalho após remover somente ocorrências extras autorizadas.
- *_SEM_DUPLICIDADES_XX.BAK: backup regenerado após o saneamento de chaves duplicadas.
- CONFLITOS_DUPLICIDADE.txt: chave, quantidade de ocorrências, campos diferentes/inconclusivos e exportação para inspeção.
- *_RECOVERY_METADATA.BAK: backup somente de metadados quando o backup completo falha.
- LOG_TECNICO_RECUPERACAO.txt: saída técnica completa das ferramentas.
- RELATORIO_RECUPERACAO.txt: resumo do diagnóstico, ações, avisos, falhas e resultado final.
- RELATORIO_SAUDE_PERFORMANCE.txt: gates pós-recuperação, índices/constraints, header, sweep, validação estrutural e contagens críticas.

7. COMO INTERPRETAR ALGUNS ERROS
"wrong page type" / "database page errors"
- Indica corrupção estrutural de página e deve ser tratada no modo avançado.

"index page errors"
- Pode indicar corrupção de índice. Índices normalmente são reconstruídos no backup/restore, mas erros combinados com páginas/BLOBs exigem mais cautela.

"BLOB errors"
- Pode haver perda localizada em campos BLOB.

"transaction ... in limbo"
- Há transação distribuída pendente. O Toolbox diagnostica, mas não decide commit/rollback automaticamente.

Falha do restore por PRIMARY/UNIQUE duplicada
- O Toolbox extrai automaticamente tabela, constraint e chave problemática do gbak.
- Em vez de insistir em -one_at_a_time, refaz o restore com gbak -inactive para manter todos os registros disponíveis.
- Se as ocorrências forem diferentes, nenhuma é descartada silenciosamente: o banco original do conflito e CONFLITOS_DUPLICIDADE.txt ficam preservados e a continuação exige confirmação.

Falha do gbak em uma tabela
- O modo avançado registra a saída verbose internamente e tenta identificar a relação afetada.
- Se o erro for compatível com índice corrompido, tenta primeiro desativar índices não vinculados a constraints somente na cópia de trabalho e repete o backup sem excluir dados.
- Se isso não resolver, em Firebird 3.0+ pode criar um backup-base com -skip_data e tentar recuperar os registros da tabela para um banco limpo.

ATALHOS DE RESULTADO
- Localizar banco: abre o Windows Explorer e seleciona somente o banco final gerado nesta execução, sem abrir o arquivo por associação.
- Abrir relatório: abre o RELATORIO_RECUPERACAO.txt quando disponível.
- Abrir pasta: abre a pasta do resultado real gerado pela rotina.

8. PÓS-RECUPERAÇÃO / SAÚDE
- O banco final só recebe APTO PARA PRODUÇÃO quando os gates críticos passam.
- O Toolbox exige acesso SQL, sweep final concluído, validação estrutural sem corrupção, zero transações em limbo e zero índices/PRIMARY KEY/UNIQUE inativos.
- Quando existe restore de teste e destino final, compara quantidades de tabelas, índices, constraints e as relações diretamente envolvidas na recuperação.
- Recuperações parciais podem ficar ESTRUTURALMENTE SAUDÁVEIS, mas continuam exigindo revisão funcional antes de produção.
- Se qualquer gate crítico falhar, o estado final é NÃO APTO PARA PRODUÇÃO e o banco não deve substituir o banco em uso.

9. LOG RESUMIDO E DETALHADO
Ver detalhes
- Mostra comandos e saída técnica ampliada para diagnóstico.

Resumir log
- Volta a mostrar apenas etapas, avisos, erros e resultado final.
- No modo avançado, o log técnico completo também é preservado em arquivo mesmo quando a tela está resumida.

10. BOAS PRÁTICAS DE SEGURANÇA
- Nunca faça recuperação pesada diretamente sobre a única cópia do banco.
- Mantenha o banco fora de uso durante validação/reparo sempre que possível.
- Não apague a pasta RECOVERY_... antes de validar o sistema e os dados recuperados.
- Após uma recuperação parcial, confira especialmente a relação indicada no relatório, as chaves não recuperadas e suas dependências.
- Um backup só deve ser considerado confiável depois de um restore de teste aprovado.

11. ONDE ENCONTRAR MAIS DETALHES
Use o botão Copiar guia completo para copiar este guia. Os arquivos docs/LEIA-ME.md e docs/MANUAL.md incluídos no pacote também documentam o funcionamento do utilitário.

Firebird Toolbox v` + appVersion + `
` + appCredit)
}

type helpSection struct {
	Title   string
	Intro   string
	Content string
}

func helpSections() []helpSection {
	return []helpSection{
		{
			Title: "Comece por aqui",
			Intro: "Fluxo recomendado para uma manutenção ou restauração segura.",
			Content: strings.TrimSpace(`1. Escolha a rotina adequada.
2. Informe a versão do Firebird que abrirá o banco de origem.
3. Se quiser restaurar, escolha a versão em "Restaurar para".
4. Selecione banco de origem, destino e pasta de backup.
5. Confira instalação do Firebird, portas e credenciais.
6. Clique em Executar. A pré-validação confere arquivo, Firebird, porta, gravação e espaço livre antes de iniciar.
7. Acompanhe Fase, Progresso, Tempo e Resultado. No término, use Localizar banco, Abrir relatório ou Abrir pasta quando esses resultados existirem.

PREFERÊNCIAS
O Toolbox lembra rotina, versões e caminhos não sensíveis entre sessões. A senha nunca é salva.

VERSÕES SUPORTADAS
Firebird 1.0, 1.5, 2.0, 2.1, 2.5, 3.0, 4.0 e 5.0.

O Toolbox procura gbak.exe, gfix.exe e gstat.exe na instalação detectada. Restaurar para versão posterior é o caminho mais comum de migração. Uma restauração para versão anterior pode falhar por incompatibilidade de metadados.

TEMPO DECORRIDO
O cronômetro começa ao confirmar a execução e é congelado no término, inclusive em erro, conflito, recuperação parcial ou cancelamento.`),
		},
		{
			Title: "Rotinas",
			Intro: "Use a rotina simples no dia a dia e a avançada apenas quando houver corrupção real.",
			Content: strings.TrimSpace(`CORRIGIR BANCO
Valida o banco, executa sweep, gera backup e, quando solicitado, restaura em outra versão. É o modo indicado para manutenção normal.

CORRIGIR BANCO (AVANÇADO)
Indicado para erros de página, BLOB, índice, falhas de gbak ou corrupção real. O Toolbox trabalha sobre cópia, faz diagnóstico, pode usar gfix -mend apenas na cópia, gera backup de recuperação e executa restore real de teste.

SOMENTE RESTAURAR
Restaura um arquivo .BAK/.FBK/.GBK diretamente para o destino usando a versão selecionada.

PULAR TABELAS (REGEX)
Disponível no modo avançado em Firebird 3.0+. Usa gbak -skip_data. O campo é opcional e deve ser usado apenas quando você realmente deseja excluir os dados correspondentes. Quando uma relação é detectada automaticamente, o skip_data pode ser apenas uma etapa intermediária de recuperação.`),
		},
		{
			Title: "Recuperação avançada",
			Intro: "O banco original é preservado; reparos destrutivos acontecem somente na cópia de trabalho.",
			Content: strings.TrimSpace(`ÍNDICES E LEITURA NATURAL
Quando a falha parece estar em índice, o Toolbox tenta preservar os dados e usa leitura NATURAL para evitar depender do índice corrompido.

CHAVES DUPLICADAS
Se o restore falhar em PRIMARY KEY/UNIQUE, a restauração é refeita com gbak -inactive. Duplicidades idênticas podem ser saneadas na cópia. Quando os registros diferem, o resolvedor mostra Registro 1 e Registro 2 e exige decisão explícita. Se aparecer OUTRA chave duplicada, o fluxo continua em loop.

RECONSTRUÇÃO LIMPA
No Firebird 3.0+, se um índice continuar inativo sem nova duplicidade detectável, o Toolbox pode reconstruir a relação em uma base limpa, copiar os registros por leitura NATURAL, reativar triggers/índices e conferir as contagens.

ARQUIVOS IMPORTANTES
RELATORIO_RECUPERACAO.txt: resumo das ações e resultado.
LOG_TECNICO_RECUPERACAO.txt: saída técnica completa.
CONFLITOS_DUPLICIDADE.txt: detalhes das chaves e registros conflitantes.
RELATORIO_SAUDE_PERFORMANCE.txt: gates finais de saúde e estrutura.

Não apague a pasta RECOVERY_... até concluir a homologação do banco recuperado.`),
		},
		{
			Title: "Resultados e saúde",
			Intro: "O resultado visual resume o estado; o relatório técnico continua sendo a fonte de confirmação.",
			Content: strings.TrimSpace(`RECUPERAÇÃO COMPLETA
Backup criado, restore de teste aprovado e nenhum dado foi excluído deliberadamente.

RECUPERAÇÃO PARCIAL
Existe resultado para inspeção, mas alguma relação não manteve todos os registros ou houve decisão explícita sobre uma duplicidade. Revise o relatório antes de produção.

BACKUP CRIADO MAS NÃO VALIDADO
O arquivo existe, porém o restore de teste não passou. Não trate esse backup como seguro.

APTO PARA PRODUÇÃO
Exige acesso SQL, sweep final, validação estrutural sem corrupção, zero transações em limbo e zero índices/PRIMARY KEY/UNIQUE inativos, além das comparações aplicáveis.

NÃO APTO PARA PRODUÇÃO
Um gate crítico falhou. Não substitua o banco em uso por esse resultado.

LOG
"Ver detalhes" amplia a saída técnica do gbak. "Resumir log" volta ao modo essencial. No modo avançado, o log técnico completo também é preservado em arquivo.`),
		},
		{
			Title: "Erros e segurança",
			Intro: "Preserve sempre o original e trate mensagens estruturais como sinal para usar o modo avançado.",
			Content: strings.TrimSpace(`WRONG PAGE TYPE / DATABASE PAGE ERRORS
Indicam corrupção estrutural de página.

INDEX PAGE ERRORS
Podem indicar corrupção de índice. Backup/restore costuma reconstruir índices, mas erros combinados exigem cautela.

BLOB ERRORS
Pode haver perda localizada em campos BLOB.

TRANSACTION ... IN LIMBO
Existe transação distribuída pendente. O Toolbox diagnostica, mas não decide commit/rollback automaticamente.

BOAS PRÁTICAS
- Nunca faça recuperação pesada sobre a única cópia do banco.
- Mantenha o banco fora de uso durante validação/reparo sempre que possível.
- Preserve a pasta RECOVERY_... até validar sistema e dados.
- Um backup só é confiável depois de restore de teste aprovado.
- Em recuperação parcial, confira as relações, chaves e dependências citadas no relatório.

Para o guia integral, use "Copiar guia completo". Os arquivos docs/LEIA-ME.md e docs/MANUAL.md do pacote também documentam o utilitário.`),
		},
	}
}

func setHelpSection(index int) {
	sections := helpSections()
	if len(sections) == 0 {
		return
	}
	if index < 0 || index >= len(sections) {
		index = 0
	}
	section := sections[index]
	if helpSectionTitleHwnd != 0 {
		procSetWindowTextW.Call(uintptr(helpSectionTitleHwnd), uintptr(unsafe.Pointer(u16(section.Title))))
	}
	if helpSectionIntroHwnd != 0 {
		procSetWindowTextW.Call(uintptr(helpSectionIntroHwnd), uintptr(unsafe.Pointer(u16(section.Intro))))
	}
	if helpContentHwnd != 0 {
		text := strings.ReplaceAll(section.Content, "\n", "\r\n")
		procSetWindowTextW.Call(uintptr(helpContentHwnd), uintptr(unsafe.Pointer(u16(text))))
		procSendMessageW.Call(uintptr(helpContentHwnd), EM_SETSEL, 0, 0)
		procSendMessageW.Call(uintptr(helpContentHwnd), EM_SCROLLCARET, 0, 0)
	}
}

func createHelpControl(exStyle uint32, className, text string, style uint32, x, y, w, h int32, id int) syscall.Handle {
	hwnd, _, _ := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(u16(className))),
		uintptr(unsafe.Pointer(u16(text))),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(helpHwnd), uintptr(id), 0, 0,
	)
	handle := syscall.Handle(hwnd)
	if guiFont != 0 {
		procSendMessageW.Call(uintptr(handle), WM_SETFONT, uintptr(guiFont), 1)
	}
	return handle
}

func helpWndProc(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) uintptr {
	switch uMsg {
	case WM_DRAWITEM:
		if lParam != 0 {
			var dis drawItemStruct
			procMoveMemory.Call(uintptr(unsafe.Pointer(&dis)), lParam, unsafe.Sizeof(dis))
			if drawOwnerButton(&dis) {
				return 1
			}
		}
	case WM_COMMAND:
		id := int(uint16(wParam & 0xffff))
		code := int(uint16((wParam >> 16) & 0xffff))
		if id == idHelpNav && code == LBN_SELCHANGE && helpNavHwnd != 0 {
			selected, _, _ := procSendMessageW.Call(uintptr(helpNavHwnd), LB_GETCURSEL, 0, 0)
			setHelpSection(int(selected))
			return 0
		}
		if code == BN_CLICKED {
			switch id {
			case idHelpCopy:
				copyTextToClipboard(helpText(), "Guia completo copiado para a área de transferência.")
				return 0
			case idHelpClose:
				procDestroyWindow.Call(uintptr(hwnd))
				return 0
			}
		}
	case WM_CLOSE:
		savePreferences(collectConfig())
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case WM_DESTROY:
		helpHwnd = 0
		helpContentHwnd = 0
		helpNavHwnd = 0
		helpSectionTitleHwnd = 0
		helpSectionIntroHwnd = 0
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(uMsg), wParam, lParam)
	return r
}

func openHelp() {
	if helpHwnd != 0 {
		procShowWindow.Call(uintptr(helpHwnd), SW_SHOW)
		procSetForegroundWindow.Call(uintptr(helpHwnd))
		return
	}

	hinst, _, _ := procGetModuleHandleW.Call(0)
	className := u16("FirebirdToolboxHelpWindow")
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(u16("Ajuda - Firebird Toolbox"))),
		WS_CAPTION|WS_SYSMENU|WS_VISIBLE,
		190, 75, uintptr(uiHelpWindowWidth), uintptr(uiHelpWindowHeight),
		uintptr(mainHwnd), 0, hinst, 0,
	)
	if hwnd == 0 {
		messageBox("Firebird Toolbox", "Não foi possível abrir a ajuda integrada.\n\nDetalhes: "+fmt.Sprint(err), MB_OK|MB_ICONERROR)
		return
	}
	helpHwnd = syscall.Handle(hwnd)

	if icon := cachedUIIcon("help", 32); icon != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, uintptr(icon))
	}
	if icon := cachedUIIcon("help", 16); icon != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, uintptr(icon))
	}

	helpIcon := createHelpControl(0, "STATIC", "", WS_CHILD|WS_VISIBLE|SS_ICON, 20, 16, 28, 28, 0)
	if icon := cachedUIIcon("help", 24); icon != 0 {
		procSendMessageW.Call(uintptr(helpIcon), STM_SETICON, uintptr(icon), 0)
	}
	title := createHelpControl(0, "STATIC", "Ajuda do Firebird Toolbox", WS_CHILD|WS_VISIBLE, 56, 12, 520, 32, 0)
	if titleFont != 0 {
		procSendMessageW.Call(uintptr(title), WM_SETFONT, uintptr(titleFont), 1)
	}
	createHelpControl(0, "STATIC", "Escolha um tópico à esquerda. O guia completo continua disponível para copiar.", WS_CHILD|WS_VISIBLE, 22, 48, 770, 22, 0)
	createHelpControl(0, "STATIC", "", WS_CHILD|WS_VISIBLE|SS_ETCHEDHORZ, 20, 74, 790, 2, 0)
	createHelpControl(0, "STATIC", "Segurança: preserve o banco original e valide sempre o restore antes de usar o resultado em produção.", WS_CHILD|WS_VISIBLE|WS_BORDER|SS_CENTERIMAGE, 20, 86, 790, 34, 0)

	navTitle := createHelpControl(0, "STATIC", "Tópicos", WS_CHILD|WS_VISIBLE, 20, 136, uiHelpNavWidth, 22, 0)
	if sectionFont != 0 {
		procSendMessageW.Call(uintptr(navTitle), WM_SETFONT, uintptr(sectionFont), 1)
	}
	helpNavHwnd = createHelpControl(WS_EX_CLIENTEDGE, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|LBS_NOTIFY, 20, 160, uiHelpNavWidth, 420, idHelpNav)
	for _, section := range helpSections() {
		procSendMessageW.Call(uintptr(helpNavHwnd), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(section.Title))))
	}
	procSendMessageW.Call(uintptr(helpNavHwnd), LB_SETCURSEL, 0, 0)

	helpSectionTitleHwnd = createHelpControl(0, "STATIC", "", WS_CHILD|WS_VISIBLE, 220, 134, 590, 26, idHelpSectionTitle)
	if sectionFont != 0 {
		procSendMessageW.Call(uintptr(helpSectionTitleHwnd), WM_SETFONT, uintptr(sectionFont), 1)
	}
	helpSectionIntroHwnd = createHelpControl(0, "STATIC", "", WS_CHILD|WS_VISIBLE, 220, 162, 590, 42, idHelpSectionIntro)
	helpContentHwnd = createHelpControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 218, 206, uiHelpContentWidth, uiHelpContentHeight, 0)
	procSendMessageW.Call(uintptr(helpContentHwnd), EM_SETMARGINS, EC_LEFTMARGIN|EC_RIGHTMARGIN, uintptr(10|(10<<16)))
	setHelpSection(0)

	createHelpControl(0, "STATIC", "Firebird Toolbox v"+appVersion+"  •  "+appCredit, WS_CHILD|WS_VISIBLE, 20, uiHelpBottomY+8, 520, 22, 0)
	createHelpControl(0, "BUTTON", "Copiar guia completo", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 548, uiHelpBottomY, 160, 32, idHelpCopy)
	createHelpControl(0, "BUTTON", "Fechar", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 718, uiHelpBottomY, 90, 32, idHelpClose)
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}

func createConflictControl(exStyle uint32, className, text string, style uint32, x, y, w, h int32, id int) syscall.Handle {
	hwnd, _, _ := procCreateWindowExW.Call(
		uintptr(exStyle), uintptr(unsafe.Pointer(u16(className))), uintptr(unsafe.Pointer(u16(text))), uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(conflictHwnd), uintptr(id), 0, 0,
	)
	handle := syscall.Handle(hwnd)
	if guiFont != 0 {
		procSendMessageW.Call(uintptr(handle), WM_SETFONT, uintptr(guiFont), 1)
	}
	return handle
}

func openConflictComparisonFile() {
	path := strings.TrimSpace(conflictResolverPath)
	if path == "" || !fileExists(path) {
		return
	}
	cmd := exec.Command("notepad.exe", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}

func conflictWndProc(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) uintptr {
	switch uMsg {
	case WM_COMMAND:
		id := int(uint16(wParam & 0xffff))
		code := int(uint16((wParam >> 16) & 0xffff))
		if code == BN_CLICKED {
			switch id {
			case idConflictKeep1:
				conflictResolverDecision = conflictResolutionKeep1
				procDestroyWindow.Call(uintptr(hwnd))
				return 0
			case idConflictKeep2:
				conflictResolverDecision = conflictResolutionKeep2
				procDestroyWindow.Call(uintptr(hwnd))
				return 0
			case idConflictExport:
				openConflictComparisonFile()
				return 0
			case idConflictCancel:
				conflictResolverDecision = conflictResolutionCancel
				procDestroyWindow.Call(uintptr(hwnd))
				return 0
			}
		}
	case WM_CLOSE:
		conflictResolverDecision = conflictResolutionCancel
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case WM_DESTROY:
		conflictHwnd = 0
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(uMsg), wParam, lParam)
	return r
}

func showDuplicateConflictResolver(conflict restoreDuplicateConflict, view duplicateResolutionView) conflictResolutionDecision {
	conflictResolverMu.Lock()
	defer conflictResolverMu.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	conflictResolverDecision = conflictResolutionCancel
	conflictResolverPath = view.FilePath
	procEnableWindow.Call(uintptr(mainHwnd), 0)
	defer func() {
		procEnableWindow.Call(uintptr(mainHwnd), 1)
		procSetForegroundWindow.Call(uintptr(mainHwnd))
		conflictResolverPath = ""
	}()

	hinst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(u16("FirebirdToolboxConflictWindow"))),
		uintptr(unsafe.Pointer(u16("Resolver conflito de registros - Firebird Toolbox"))),
		WS_CAPTION|WS_SYSMENU|WS_VISIBLE,
		90, 45, 1010, 800,
		uintptr(mainHwnd), 0, hinst, 0,
	)
	if hwnd == 0 {
		messageBox("Firebird Toolbox", "Não foi possível abrir o resolvedor de conflitos.\n\nDetalhes: "+fmt.Sprint(err), MB_OK|MB_ICONERROR)
		return conflictResolutionCancel
	}
	conflictHwnd = syscall.Handle(hwnd)
	if icon := cachedUIIcon("database", 32); icon != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, uintptr(icon))
	}
	if icon := cachedUIIcon("database", 16); icon != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, uintptr(icon))
	}

	title := createConflictControl(0, "STATIC", "Conflito de chave: escolha qual registro deve permanecer", WS_CHILD|WS_VISIBLE, 22, 16, 930, 30, 0)
	if titleFont != 0 {
		procSendMessageW.Call(uintptr(title), WM_SETFONT, uintptr(titleFont), 1)
	}
	createConflictControl(0, "STATIC", "Tabela "+conflict.Relation+" | "+duplicateConflictKeyLabel(conflict), WS_CHILD|WS_VISIBLE, 24, 50, 940, 22, 0)

	summary := createConflictControl(WS_EX_CLIENTEDGE, "EDIT", view.SummaryText, WS_CHILD|WS_VISIBLE|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 22, 80, 950, 112, 0)
	procSendMessageW.Call(uintptr(summary), EM_SETSEL, 0, 0)
	createConflictControl(0, "STATIC", "Registro 1", WS_CHILD|WS_VISIBLE, 22, 202, 200, 22, 0)
	createConflictControl(0, "STATIC", "Registro 2", WS_CHILD|WS_VISIBLE, 506, 202, 200, 22, 0)
	left := createConflictControl(WS_EX_CLIENTEDGE, "EDIT", view.Record1Text, WS_CHILD|WS_VISIBLE|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 22, 226, 466, 370, 0)
	right := createConflictControl(WS_EX_CLIENTEDGE, "EDIT", view.Record2Text, WS_CHILD|WS_VISIBLE|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 506, 226, 466, 370, 0)
	if monoFont != 0 {
		procSendMessageW.Call(uintptr(left), WM_SETFONT, uintptr(monoFont), 1)
		procSendMessageW.Call(uintptr(right), WM_SETFONT, uintptr(monoFont), 1)
	}
	procSendMessageW.Call(uintptr(left), EM_SETSEL, 0, 0)
	procSendMessageW.Call(uintptr(right), EM_SETSEL, 0, 0)
	createConflictControl(0, "STATIC", view.DetailsText, WS_CHILD|WS_VISIBLE, 22, 610, 950, 60, 0)
	createConflictControl(0, "BUTTON", "Manter registro 1", WS_CHILD|WS_VISIBLE|WS_TABSTOP, 22, 688, 170, 34, idConflictKeep1)
	createConflictControl(0, "BUTTON", "Manter registro 2", WS_CHILD|WS_VISIBLE|WS_TABSTOP, 202, 688, 170, 34, idConflictKeep2)
	createConflictControl(0, "BUTTON", "Abrir comparação", WS_CHILD|WS_VISIBLE|WS_TABSTOP, 602, 688, 170, 34, idConflictExport)
	createConflictControl(0, "BUTTON", "Preservar e cancelar", WS_CHILD|WS_VISIBLE|WS_TABSTOP, 782, 688, 190, 34, idConflictCancel)
	createConflictControl(0, "STATIC", "Nenhuma opção altera o banco original. A escolha afeta somente a cópia de trabalho.", WS_CHILD|WS_VISIBLE, 22, 735, 950, 22, 0)

	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
	procSetForegroundWindow.Call(hwnd)
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return conflictResolverDecision
}

func resolveResultFolder(paths ...string) string {
	clean := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, raw := range paths {
		path := strings.Trim(strings.TrimSpace(raw), `"`)
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		key := strings.ToLower(path)
		if seen[key] {
			continue
		}
		seen[key] = true
		clean = append(clean, path)
	}

	// Primeiro prioriza resultados que realmente existem. Isso evita abrir a
	// pasta de um destino apenas sugerido quando a rotina terminou com relatório
	// ou backup em outro local.
	for _, path := range clean {
		if dirExists(path) {
			return path
		}
		if fileExists(path) {
			if dir := filepath.Dir(path); dirExists(dir) {
				return dir
			}
		}
	}

	// Antes da primeira execução é útil abrir a pasta pai de um arquivo de
	// destino ainda não criado.
	for _, path := range clean {
		dir := filepath.Dir(path)
		if dir != "." && dirExists(dir) {
			return dir
		}
	}
	return ""
}

func openDirectoryNative(dir string) error {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "" || !dirExists(dir) {
		return errors.New("a pasta informada não existe")
	}

	// ShellExecute usa o shell do próprio Windows e funciona melhor com caminhos
	// Unicode, UNC e associações do Explorer do que iniciar explorer.exe às cegas.
	r, _, _ := procShellExecuteW.Call(
		uintptr(mainHwnd),
		uintptr(unsafe.Pointer(u16("open"))),
		uintptr(unsafe.Pointer(u16(dir))),
		0,
		0,
		SW_SHOWNORMAL,
	)
	if r > 32 {
		return nil
	}

	// Fallback para instalações do Windows em que o shell retorna um código de
	// erro, mas o Explorer continua disponível diretamente.
	cmd := exec.Command("explorer.exe", dir)
	if err := cmd.Start(); err == nil {
		return nil
	}
	return fmt.Errorf("o Windows não conseguiu abrir a pasta (ShellExecute=%d)", r)
}

func openFileNative(path string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || !fileExists(path) {
		return errors.New("o arquivo informado não existe")
	}
	r, _, _ := procShellExecuteW.Call(
		uintptr(mainHwnd),
		uintptr(unsafe.Pointer(u16("open"))),
		uintptr(unsafe.Pointer(u16(path))),
		0,
		0,
		SW_SHOWNORMAL,
	)
	if r > 32 {
		return nil
	}
	return fmt.Errorf("o Windows não conseguiu abrir o arquivo (ShellExecute=%d)", r)
}

func updateResultButtons() {
	_, _, report := runResultSnapshot()
	enableControl(idOpenReport, fileExists(report))
	dest := strings.TrimSpace(getText(idDest))
	if fileGeneratedThisRun(dest) {
		setLastDatabase(dest)
	} else {
		setLastDatabase("")
	}
	enableControl(idOpenDatabase, fileExists(lastDatabaseSnapshot()))
}

func openResultReport() {
	_, _, report := runResultSnapshot()
	if report == "" || !fileExists(report) {
		messageBox("Firebird Toolbox", "Ainda não há um relatório de recuperação disponível.", MB_OK|MB_ICONINFO)
		return
	}
	if err := openFileNative(report); err != nil {
		messageBox("Firebird Toolbox", "Não foi possível abrir o relatório.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
		return
	}
	setStatus("Relatório de recuperação aberto.")
}

func openResultDatabase() {
	dest := lastDatabaseSnapshot()
	if dest == "" || !fileExists(dest) {
		messageBox("Firebird Toolbox", "Ainda não há um banco final gerado nesta execução para localizar.", MB_OK|MB_ICONINFO)
		return
	}
	// Bancos Firebird não devem ser executados/abertos por associação. O botão
	// abre o Explorer já selecionando exatamente o banco final desta execução.
	args := []string{"/select," + dest}
	cmd := exec.Command("explorer.exe", args...)
	if err := cmd.Start(); err != nil {
		if err2 := openDirectoryNative(filepath.Dir(dest)); err2 != nil {
			messageBox("Firebird Toolbox", "Não foi possível localizar o banco final.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
			return
		}
	}
	setStatus("Banco final localizado e selecionado no Windows Explorer.")
}

func openDestination() {
	_, backup, report := runResultSnapshot()
	dir := resolveResultFolder(
		strings.TrimSpace(getText(idDest)),
		report,
		backup,
		strings.TrimSpace(getText(idBackupDir)),
	)
	if dir == "" {
		messageBox("Firebird Toolbox", "Ainda não há uma pasta de resultado disponível. Selecione um destino ou execute uma rotina primeiro.", MB_OK|MB_ICONINFO)
		return
	}
	if err := openDirectoryNative(dir); err != nil {
		messageBox("Firebird Toolbox", "Não foi possível abrir a pasta de resultado.\n\n"+dir+"\n\n"+err.Error(), MB_OK|MB_ICONERROR)
		appendLog("Falha ao abrir pasta de resultado: " + dir + " - " + err.Error())
		return
	}
	setStatus("Pasta de resultado aberta no Windows Explorer.")
}

func wndProc(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) uintptr {
	switch uMsg {
	case WM_CREATE:
		return 0
	case WM_DRAWITEM:
		if lParam != 0 {
			var dis drawItemStruct
			procMoveMemory.Call(uintptr(unsafe.Pointer(&dis)), lParam, unsafe.Sizeof(dis))
			if drawOwnerButton(&dis) {
				return 1
			}
		}
	case WM_CTLCOLORSTATIC:
		if syscall.Handle(lParam) == controls[idLog] && logBrush != 0 {
			procSetTextColor.Call(wParam, rgb(226, 232, 240))
			procSetBkColor.Call(wParam, rgb(28, 31, 36))
			return uintptr(logBrush)
		}
		if syscall.Handle(lParam) == controls[idResult] {
			text := strings.ToUpper(getText(idResult))
			brush := resultBrushNeutral
			fg := rgb(51, 65, 85)
			bg := rgb(248, 250, 252)
			switch {
			case strings.Contains(text, "NÃO APTO") || strings.Contains(text, "ERRO"):
				brush, fg, bg = resultBrushError, rgb(153, 27, 27), rgb(254, 242, 242)
			case strings.Contains(text, "PARCIAL") || strings.Contains(text, "CONFLITO") || strings.Contains(text, "INTERROMPIDO"):
				brush, fg, bg = resultBrushWarning, rgb(146, 64, 14), rgb(255, 251, 235)
			case strings.Contains(text, "✓") || strings.Contains(text, "CONCLUÍDO") || strings.Contains(text, "APTO PARA PRODUÇÃO"):
				brush, fg, bg = resultBrushSuccess, rgb(22, 101, 52), rgb(240, 253, 244)
			}
			procSetTextColor.Call(wParam, fg)
			procSetBkColor.Call(wParam, bg)
			if brush != 0 {
				return uintptr(brush)
			}
		}

		// Demais textos estáticos usam o mesmo branco da janela. Isso remove o
		// fundo cinza sem usar transparência, evitando rastros/duplicações quando
		// o Windows repinta textos que mudam durante a execução.
		if enabled, _, _ := procIsWindowEnabled.Call(lParam); enabled == 0 {
			procSetTextColor.Call(wParam, rgb(148, 163, 184))
		} else {
			procSetTextColor.Call(wParam, rgb(17, 24, 39))
		}
		procSetBkColor.Call(wParam, rgb(255, 255, 255))
		if staticBrushWhite != 0 {
			return uintptr(staticBrushWhite)
		}
	case WM_COMMAND:
		id := int(uint16(wParam & 0xffff))
		code := int(uint16((wParam >> 16) & 0xffff))
		if code == CBN_SELCHANGE {
			switch id {
			case idRoutine:
				routineChanged()
			case idSourceVersion:
				sourceVersionChanged()
			case idRestoreVersion:
				restoreVersionChanged()
			}
			return 0
		}
		if code == BN_CLICKED {
			switch id {
			case idSourceBtn:
				browseSource()
			case idDestBtn:
				browseDestination()
			case idBackupDirBtn:
				browseBackupDir()
			case idFbSourceBtn:
				browseFB(false)
			case idFbTargetBtn:
				browseFB(true)
			case idPasswordToggle:
				togglePassword()
			case idExecute:
				executeWorkflow()
			case idStop:
				stopWorkflow()
			case idOpenDest:
				openDestination()
			case idCopyPath:
				copyDestinationPath()
			case idHelp:
				openHelp()
			case idOpenReport:
				openResultReport()
			case idOpenDatabase:
				openResultDatabase()
			case idClearLog:
				clearLog()
			case idCopyLog:
				copyLogToClipboard()
			case idDetailedLog:
				toggleDetailedLog()
			}
		}
		return 0
	case WM_CLOSE:
		if isRunning() {
			if messageBox("Firebird Toolbox", "Existe uma rotina em execução.\n\nDeseja interromper e fechar o programa?", MB_YESNO|MB_ICONWARNING) != IDYES {
				return 0
			}
			runMu.Lock()
			if currentCancel != nil {
				currentCancel()
			}
			runMu.Unlock()
		}
		savePreferences(collectConfig())
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(uMsg), wParam, lParam)
	return r
}

func addCombo(id int, items []string, x, y, w int32, selected int) syscall.Handle {
	combo := createControl(WS_EX_CLIENTEDGE, "COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|CBS_HASSTRINGS, x, y, w, 220, id)
	for _, item := range items {
		procSendMessageW.Call(uintptr(combo), CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(item))))
	}
	procSendMessageW.Call(uintptr(combo), CB_SETCURSEL, uintptr(selected), 0)
	return combo
}

func buildArchLabel() string {
	if runtime.GOARCH == "386" {
		return "Windows x86/x64"
	}
	return "Windows x64 (dev)"
}

func buildUI() {
	addIcon("flame", 20, 14, 28)
	createControl(0, "STATIC", "Firebird Toolbox", WS_CHILD|WS_VISIBLE, 56, 10, 330, 30, idHeaderTitle)
	setControlFont(idHeaderTitle, titleFont)
	createControl(0, "STATIC", "Firebird 1.0 a 5.0 • manutenção, recuperação e restauração segura", WS_CHILD|WS_VISIBLE, 56, 39, 430, 20, 3002)
	createControl(0, "STATIC", "v"+appVersion+"  •  "+buildArchLabel(), WS_CHILD|WS_VISIBLE, 744, 22, 160, 20, 3004)

	addSectionBox(16, 66, 888, 106)
	addSectionHeader(idSectionRoutine, "gear", "1. Rotina e versão", 30, 72)
	createControl(0, "STATIC", "Rotina:", WS_CHILD|WS_VISIBLE, 30, 106, 62, 20, 3003)
	addCombo(idRoutine, routineNames(routines), 96, 102, 270, 0)
	createControl(0, "STATIC", "Versão da rotina:", WS_CHILD|WS_VISIBLE, 386, 106, 120, 20, lblSourceVer)
	addCombo(idSourceVersion, versions, 510, 102, 95, versionIndex("5.0"))
	addIcon("restore", 622, 104, 16)
	createControl(0, "STATIC", "Restaurar para:", WS_CHILD|WS_VISIBLE, 646, 106, 90, 20, lblRestoreVer)
	addCombo(idRestoreVersion, restoreVersions, 738, 102, 146, restoreVersionIndex("5.0"))
	// Uma linha so: a descricao de cada rotina precisa caber em uiDescriptionWidth.
	createControl(0, "STATIC", routines[0].Description, WS_CHILD|WS_VISIBLE, 30, 140, uiDescriptionWidth, 20, idDescription)

	addSectionBox(16, 182, 888, 140)
	addSectionHeader(idSectionFiles, "folder", "2. Origem e destino", 30, 188)
	createControl(0, "STATIC", "Banco de origem:", WS_CHILD|WS_VISIBLE, 30, 220, uiFilesLabelWidth, 20, lblSource)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, uiFilesFieldX, 216, 778-uiFilesFieldX, 26, idSource)
	createIconButton(idSourceBtn, "Procurar", 790, 215, 94, 28)

	createControl(0, "STATIC", "Arquivo de destino:", WS_CHILD|WS_VISIBLE, 30, 255, uiFilesLabelWidth, 20, lblDest)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, uiFilesFieldX, 251, 778-uiFilesFieldX, 26, idDest)
	createIconButton(idDestBtn, "Procurar", 790, 250, 94, 28)

	createControl(0, "STATIC", "Pasta do backup:", WS_CHILD|WS_VISIBLE, 30, 290, uiFilesLabelWidth, 20, lblBackup)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, uiFilesFieldX, 286, 778-uiFilesFieldX, 26, idBackupDir)
	createIconButton(idBackupDirBtn, "Procurar", 790, 285, 94, 28)

	addSectionBox(16, 332, 888, 168)
	addSectionHeader(idSectionFirebird, "database", "3. Configurações do Firebird", 30, 338)
	createControl(0, "STATIC", "Firebird da rotina:", WS_CHILD|WS_VISIBLE, 30, 372, uiFirebirdLabelWidth, 20, lblFbSource)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, uiFirebirdFieldX, 368, 682-uiFirebirdFieldX, 26, idFbSource)
	createIconButton(idFbSourceBtn, "Escolher", 694, 367, 86, 28)
	createControl(0, "STATIC", "Porta:", WS_CHILD|WS_VISIBLE, 792, 372, 42, 20, lblSourcePort)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, 836, 368, 48, 26, idSourcePort)

	createControl(0, "STATIC", "Firebird da restauração:", WS_CHILD|WS_VISIBLE, 30, 407, uiFirebirdLabelWidth, 20, lblFbTarget)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, uiFirebirdFieldX, 403, 682-uiFirebirdFieldX, 26, idFbTarget)
	createIconButton(idFbTargetBtn, "Escolher", 694, 402, 86, 28)
	createControl(0, "STATIC", "Porta:", WS_CHILD|WS_VISIBLE, 792, 407, 42, 20, lblTargetPort)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, 836, 403, 48, 26, idTargetPort)

	createControl(0, "STATIC", "Serviço:", WS_CHILD|WS_VISIBLE, 30, 447, 60, 20, lblService)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, 92, 443, 202, 26, idService)
	createControl(0, "STATIC", "Usuário:", WS_CHILD|WS_VISIBLE, 312, 447, 55, 20, 3010)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "SYSDBA", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, 370, 443, 144, 26, idUser)
	createControl(0, "STATIC", "Senha:", WS_CHILD|WS_VISIBLE, 530, 447, 50, 20, 3011)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "masterkey", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL|ES_PASSWORD, 582, 443, 166, 26, idPassword)
	createIconButton(idPasswordToggle, "Mostrar senha", 760, 442, 124, 28)

	addSectionBox(16, 510, 888, 204)
	addSectionHeader(idSectionExecute, "play", "4. Executar", 30, 516)
	createIconButton(idExecute, "Executar", 30, 544, 124, 32)
	createIconButton(idStop, "Parar", 164, 544, 94, 32)
	createIconButton(idOpenDest, "Abrir pasta", 268, 544, 124, 32)
	createIconButton(idCopyPath, "Copiar caminho", 402, 544, 134, 32)
	createIconButton(idHelp, "Ajuda", 546, 544, 96, 32)

	// Os atalhos de resultado ficam em uma segunda linha. Assim os textos completos
	// permanecem legíveis sem apertar as ações principais nem aumentar a janela.
	createControl(0, "STATIC", "Pular tabelas (avançado):", WS_CHILD|WS_VISIBLE, 30, 586, 158, 20, lblSkipData)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, uiSkipEditX, uiResultShortcutY, uiSkipEditWidth, 26, idSkipData)
	createIconButton(idOpenReport, "Abrir relatório", uiOpenReportX, uiResultShortcutY, uiOpenReportWidth, 28)
	createIconButton(idOpenDatabase, "Localizar banco", uiLocateDatabaseX, uiResultShortcutY, uiLocateDatabaseWidth, 28)

	// Andamento e resultado ocupam linhas próprias para manter textos longos
	// integralmente legíveis sem disputar espaço com a barra de progresso.
	createControl(0, "STATIC", "Etapa:", WS_CHILD|WS_VISIBLE, 30, 616, 70, 20, 3022)
	createControl(0, "STATIC", "Preparando...", WS_CHILD|WS_VISIBLE, 102, 616, uiProgressTextWidth, 20, idProgressText)

	createControl(0, "STATIC", "Progresso:", WS_CHILD|WS_VISIBLE, 30, 642, 70, 20, 3020)
	progress := createControl(0, "msctls_progress32", "", WS_CHILD|WS_VISIBLE|PBS_SMOOTH, 102, 641, uiProgressBarWidth, 18, idProgress)
	procSendMessageW.Call(uintptr(progress), PBM_SETRANGE32, 0, 100)

	createControl(0, "STATIC", "Tempo decorrido:", WS_CHILD|WS_VISIBLE, 30, 666, 105, 20, 3024)
	createControl(0, "STATIC", formatElapsedDuration(0), WS_CHILD|WS_VISIBLE, 138, 666, uiElapsedTextWidth, 20, idElapsed)

	createControl(0, "STATIC", "Resultado:", WS_CHILD|WS_VISIBLE, 30, 690, 70, 20, 3023)
	createControl(0, "STATIC", "Aguardando execução", WS_CHILD|WS_VISIBLE|WS_BORDER|SS_CENTERIMAGE, 102, 684, 780, 26, idResult)
	setControlFont(idResult, sectionFont)

	addSectionBox(16, 720, 888, 170)
	addSectionHeader(idSectionLog, "terminal", "Log de execução", 30, 726)
	createIconButton(idDetailedLog, "Ver detalhes", 522, 726, 130, 28)
	createIconButton(idClearLog, "Limpar", 662, 726, 102, 28)
	createIconButton(idCopyLog, "Copiar", 774, 726, 110, 28)
	createControl(WS_EX_CLIENTEDGE, "EDIT", "", WS_CHILD|WS_VISIBLE|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY|WS_VSCROLL, 28, 760, 864, uiLogEditHeight, idLog)
	setControlFont(idLog, monoFont)

	createControl(0, "STATIC", "Status:", WS_CHILD|WS_VISIBLE, 22, 899, 45, 20, 3012)
	createControl(0, "STATIC", "Pronto. Selecione a rotina, os arquivos e clique em Executar.", WS_CHILD|WS_VISIBLE, 70, 899, 500, 20, idStatus)
	createControl(0, "STATIC", appCredit, WS_CHILD|WS_VISIBLE, 590, 899, 300, 20, 3013)

	setText(idFbSource, detectFirebird("5.0"))
	setText(idFbTarget, detectFirebird("5.0"))
	setText(idService, defaults["5.0"].Service)
	setText(idSourcePort, detectFirebirdPort(getText(idFbSource), "5.0"))
	setText(idTargetPort, detectFirebirdPort(getText(idFbTarget), "5.0"))
	enableControl(idStop, false)
	enableControl(idOpenReport, false)
	enableControl(idOpenDatabase, false)
	applyPasswordVisibility(false)
	setProgressState(PBST_NORMAL)
	setProgress(0, "Aguardando...")
	routineChanged()
	applySavedPreferences()
}

func createFont(height int32, weight int32, face string) syscall.Handle {
	h, _, _ := procCreateFontW.Call(
		uintptr(height), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(u16(face))),
	)
	return syscall.Handle(h)
}

func showStartupFailure(v any) {
	message := fmt.Sprintf("O Firebird Toolbox encontrou um erro ao iniciar e nao pode continuar.\n\nDetalhes: %v\n\nEsta mensagem evita que o programa feche silenciosamente.", v)
	messageBox("Firebird Toolbox - erro de inicializacao", message, MB_OK|MB_ICONERROR)
}

func main() {
	defer func() {
		if recovered := recover(); recovered != nil {
			showStartupFailure(recovered)
		}
	}()
	runtime.LockOSThread()
	procCoInitializeEx.Call(0, 2)
	procInitCommonControls.Call()
	defer procCoUninitialize.Call()
	hinst, _, _ := procGetModuleHandleW.Call(0)
	cursor, _, _ := procLoadCursorW.Call(0, IDC_ARROW)
	appIconSmall = loadAppResourceIcon(syscall.Handle(hinst), 16)
	appIconBig = loadAppResourceIcon(syscall.Handle(hinst), 32)
	if appIconSmall != 0 {
		liveIcons = append(liveIcons, appIconSmall)
	}
	if appIconBig != 0 && appIconBig != appIconSmall {
		liveIcons = append(liveIcons, appIconBig)
	}
	guiFont = createFont(-15, FW_NORMAL, "Segoe UI")
	titleFont = createFont(-24, FW_SEMIBOLD, "Segoe UI")
	sectionFont = createFont(-15, FW_SEMIBOLD, "Segoe UI")
	monoFont = createFont(-15, FW_NORMAL, "Consolas")
	brush, _, _ := procCreateSolidBrush.Call(rgb(28, 31, 36))
	logBrush = syscall.Handle(brush)
	whiteBrush, _, _ := procCreateSolidBrush.Call(rgb(255, 255, 255))
	staticBrushWhite = syscall.Handle(whiteBrush)
	neutralBrush, _, _ := procCreateSolidBrush.Call(rgb(248, 250, 252))
	resultBrushNeutral = syscall.Handle(neutralBrush)
	successBrush, _, _ := procCreateSolidBrush.Call(rgb(240, 253, 244))
	resultBrushSuccess = syscall.Handle(successBrush)
	warningBrush, _, _ := procCreateSolidBrush.Call(rgb(255, 251, 235))
	resultBrushWarning = syscall.Handle(warningBrush)
	errorBrush, _, _ := procCreateSolidBrush.Call(rgb(254, 242, 242))
	resultBrushError = syscall.Handle(errorBrush)

	className := u16("FirebirdToolboxWindow")
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     syscall.Handle(hinst),
		hIcon:         appIconBig,
		hCursor:       syscall.Handle(cursor),
		hbrBackground: syscall.Handle(COLOR_WINDOW + 1),
		lpszClassName: className,
		hIconSm:       appIconSmall,
	}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		showStartupFailure(fmt.Sprintf("nao foi possivel registrar a janela: %v", err))
		return
	}

	helpClassName := u16("FirebirdToolboxHelpWindow")
	helpClass := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   syscall.NewCallback(helpWndProc),
		hInstance:     syscall.Handle(hinst),
		hIcon:         appIconBig,
		hCursor:       syscall.Handle(cursor),
		hbrBackground: syscall.Handle(COLOR_WINDOW + 1),
		lpszClassName: helpClassName,
		hIconSm:       appIconSmall,
	}
	if helpAtom, _, helpErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&helpClass))); helpAtom == 0 {
		showStartupFailure(fmt.Sprintf("nao foi possivel registrar a janela de ajuda: %v", helpErr))
		return
	}

	conflictClassName := u16("FirebirdToolboxConflictWindow")
	conflictClass := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   syscall.NewCallback(conflictWndProc),
		hInstance:     syscall.Handle(hinst),
		hIcon:         appIconBig,
		hCursor:       syscall.Handle(cursor),
		hbrBackground: syscall.Handle(COLOR_WINDOW + 1),
		lpszClassName: conflictClassName,
		hIconSm:       appIconSmall,
	}
	if conflictAtom, _, conflictErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&conflictClass))); conflictAtom == 0 {
		showStartupFailure(fmt.Sprintf("nao foi possivel registrar o resolvedor de conflitos: %v", conflictErr))
		return
	}

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(u16("Firebird Toolbox"))),
		WS_CAPTION|WS_SYSMENU|WS_MINIMIZEBOX,
		120, 40, 930, 966,
		0, 0, hinst, 0,
	)
	if hwnd == 0 {
		return
	}
	mainHwnd = syscall.Handle(hwnd)
	setAppIcons()
	buildUI()
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	cleanupUIResources()

	if guiFont != 0 {
		procDeleteObject.Call(uintptr(guiFont))
	}
	if titleFont != 0 {
		procDeleteObject.Call(uintptr(titleFont))
	}
	if sectionFont != 0 {
		procDeleteObject.Call(uintptr(sectionFont))
	}
	if monoFont != 0 {
		procDeleteObject.Call(uintptr(monoFont))
	}
	if logBrush != 0 {
		procDeleteObject.Call(uintptr(logBrush))
	}
	for _, brush := range []syscall.Handle{staticBrushWhite, resultBrushNeutral, resultBrushSuccess, resultBrushWarning, resultBrushError} {
		if brush != 0 {
			procDeleteObject.Call(uintptr(brush))
		}
	}
}
