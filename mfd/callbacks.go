package mfd

import "github.com/rs/zerolog/log"

const (
	softButton_Select = 0x00000001 // X52Pro ScrollClick
	softButton_Up     = 0x00000002 // X52Pro ScrollUp, FIP RightScrollClockwize
	softButton_Down   = 0x00000004 // X52Pro ScrollDown, FIP RightScrollAnticlockwize
)

// onEnumerate is called if a device is plugged in when the enumerate function is called.
func onEnumerate(hdevice uintptr, context uintptr) uintptr {
	log.Debug().Msg("Found device")
	device = hdevice
	initPages()
	return S_OK
}

// onDeviceChanged is called whenever a device is plugged in or removed
func onDeviceChanged(hdevice uintptr, added bool, context uintptr) uintptr {
	log.Trace().Bool("added", added).Msg("onDeviceChanged")
	if added {
		log.Debug().Msg("New device was plugged in")
		device = hdevice
		initPages()
	} else {
		device = 0
		log.Warn().Msg("Device was unplugged. You should restart this program.")
	}

	return S_OK
}

// onPageChange is called whenever the page scroll wheel is used.
// The current (or last active) page is passed in the page parameter
// The setActive flag indicates whether or not the new page is active (false if the profile page is set)
func onPageChange(hdevice uintptr, page uint32, setActive bool, context uintptr) uintptr {
	log.Trace().Uint32("page", page).Bool("setActive", setActive).Msg("onPageChange")
	currentPage = page
	pageActive = setActive
	refreshDisplay()

	return S_OK
}

// onSoftButton is called when the right scroll wheel is rolled or clicked
func onSoftButton(hdevice uintptr, buttons uint32, context uintptr) uintptr {
	log.Trace().Uint32("buttons", buttons).Msg("onSoftbutton")
	switch buttons {
	case softButton_Select:
		if buttonCallback != nil {
			buttonCallback()
		}
	case softButton_Up:
		decrementLine()
	case softButton_Down:
		incrementLine()
	}
	return S_OK
}
