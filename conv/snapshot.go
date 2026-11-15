package conv

import (
	"context"
	"fmt"
	"math"
)

// SnapshotAt saves the frame at the seek position t (see FrameSeek) as a
// full-size PNG "<name>-snapshot.png" beside the video, then
// "<name>-snapshot (1).png" and so on: nothing is ever replaced. It is the
// same frame the Video Snapshots window shows for that position.
func SnapshotAt(ctx context.Context, path string, t float64) (string, error) {
	dst := uniquePath(path, "-snapshot", ".png")
	return ffmpegPre(ctx, []string{"-ss", fmt.Sprintf("%.3f", math.Max(0, t))}, path, dst, []string{"-frames:v", "1", "-an", "-sn"}, 0, nil)
}
