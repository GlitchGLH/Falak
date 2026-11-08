package conv

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// On Windows the move and the "is the name free" check are one step:
// MoveFileEx without MOVEFILE_REPLACE_EXISTING fails if the name exists
// (os.Rename would replace a file there).
func init() {
	renameNoReplace = func(from, to string) error {
		f, err := windows.UTF16PtrFromString(from)
		if err != nil {
			return err
		}
		t, err := windows.UTF16PtrFromString(to)
		if err != nil {
			return err
		}
		if err := windows.MoveFileEx(f, t, 0); err != nil {
			if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_FILE_EXISTS) {
				err = os.ErrExist
			}
			return &os.LinkError{Op: "rename", Old: from, New: to, Err: err}
		}
		return nil
	}
}
