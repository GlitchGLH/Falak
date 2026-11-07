//go:build !windows

package conv

import (
	"context"
	"errors"
)

func openPDF(path string) (pdfDoc, error) {
	return nil, errors.New("PDF input is only available on Windows")
}

func mergePDFs(ctx context.Context, files []string, progress func(float64)) ([]byte, error) {
	return nil, errors.New("PDF input is only available on Windows")
}
