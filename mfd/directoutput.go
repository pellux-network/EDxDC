package mfd

import (
	"syscall"
	"unsafe"

	"github.com/rs/zerolog/log"
)

const (
	// HRESULT codes
	S_OK             = 0x00000000
	E_PAGENOTACTIVE  = 0xFF040001
	E_BUFFERTOOSMALL = 0xFF040000 | uintptr(syscall.ERROR_BUFFER_OVERFLOW)

	FLAG_SET_AS_ACTIVE = 0x00000001
)

const (
	dllPath    = "./bin/DirectOutput.dll"
	pluginName = "EDxDC"
	context    = 0xCAFEBABE
)

var directOutput = syscall.NewLazyDLL(dllPath)

func initialize() {
	pluginNamePtr, _ := syscall.UTF16PtrFromString(pluginName)
	callProc("DirectOutput_Initialize", uintptr(unsafe.Pointer(pluginNamePtr)))
}

func deinitialize() {
	callProc("DirectOutput_Deinitialize")
}

func enumerate() {
	callback := syscall.NewCallback(onEnumerate)
	callProc("DirectOutput_Enumerate", callback, context)
}

func registerDeviceCallback() {
	callback := syscall.NewCallback(onDeviceChanged)
	callProc("DirectOutput_RegisterDeviceCallback", callback, context)
}

func registerPageCallback(device uintptr) {
	callback := syscall.NewCallback(onPageChange)
	callProc("DirectOutput_RegisterPageCallback", device, callback, context)
}

func registerSoftButtonCallback(device uintptr) {
	callback := syscall.NewCallback(onSoftButton)
	callProc("DirectOutput_RegisterSoftButtonCallback", device, callback, context)
}

func addPage(pageNumber uint32, active bool) {
	var flag uintptr = 0
	if active {
		flag = uintptr(FLAG_SET_AS_ACTIVE)
	}
	callProc("DirectOutput_AddPage", device, uintptr(pageNumber), flag)
}

func setString(page, lineIdx uint32, line string) {
	linePtr, _ := syscall.UTF16PtrFromString(line)
	lineLen := uintptr(len(line))
	callProc("DirectOutput_SetString", device, uintptr(page), uintptr(lineIdx), lineLen, uintptr(unsafe.Pointer(linePtr)))
}

func callProc(procname string, args ...uintptr) {
	proc := directOutput.NewProc(procname)
	hresult, _, err := proc.Call(args...)

	switch hresult {
	case S_OK:
		return
	case E_PAGENOTACTIVE:
		log.Warn().Uint64("hresult", uint64(hresult)).Msg("E_PAGENOTACTIVE")
	default:
		log.Warn().Uint64("hresult", uint64(hresult)).Msg("DirectOutput call failed")
		log.Fatal().Err(err).Msg("DirectOutput fatal error")
	}
}
