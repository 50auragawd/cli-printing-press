package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDir(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, src string)
		check func(t *testing.T, src, dst string)
	}{
		{
			name: "regular files and directories",
			setup: func(t *testing.T, src string) {
				require.NoError(t, os.MkdirAll(filepath.Join(src, "sub"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(src, "root.txt"), []byte("root"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "sub", "nested.txt"), []byte("nested"), 0o644))
			},
			check: func(t *testing.T, _, dst string) {
				assert.FileExists(t, filepath.Join(dst, "root.txt"))
				assert.FileExists(t, filepath.Join(dst, "sub", "nested.txt"))
				data, err := os.ReadFile(filepath.Join(dst, "root.txt"))
				require.NoError(t, err)
				assert.Equal(t, "root", string(data))
			},
		},
		{
			name: "internal file symlink preserved as symlink",
			setup: func(t *testing.T, src string) {
				target := filepath.Join(src, "target.txt")
				require.NoError(t, os.WriteFile(target, []byte("target"), 0o644))
				require.NoError(t, os.Symlink("target.txt", filepath.Join(src, "link.txt")))
			},
			check: func(t *testing.T, _, dst string) {
				linkPath := filepath.Join(dst, "link.txt")
				info, err := os.Lstat(linkPath)
				require.NoError(t, err)
				assert.NotZero(t, info.Mode()&os.ModeSymlink, "expected symlink, got regular file")
			},
		},
		{
			name: "internal directory symlink preserved as symlink",
			setup: func(t *testing.T, src string) {
				targetDir := filepath.Join(src, "target-dir")
				require.NoError(t, os.MkdirAll(targetDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(targetDir, "big.bin"), []byte("data"), 0o644))
				require.NoError(t, os.Symlink("target-dir", filepath.Join(src, "linked-dir")))
			},
			check: func(t *testing.T, _, dst string) {
				linkPath := filepath.Join(dst, "linked-dir")
				info, err := os.Lstat(linkPath)
				require.NoError(t, err)
				assert.NotZero(t, info.Mode()&os.ModeSymlink,
					"directory symlink should be preserved as a symlink, not followed")
				targetData, err := os.ReadFile(filepath.Join(dst, "target-dir", "big.bin"))
				require.NoError(t, err)
				assert.Equal(t, "data", string(targetData))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "src")
			dst := filepath.Join(t.TempDir(), "dst")
			require.NoError(t, os.MkdirAll(src, 0o755))

			tt.setup(t, src)
			require.NoError(t, CopyDir(src, dst))
			tt.check(t, src, dst)
		})
	}
}

func TestCopyDirRejectsExternalSymlinks(t *testing.T) {
	tests := []struct {
		name       string
		linkName   string
		linkTarget string
	}{
		{
			name:       "absolute external target",
			linkName:   "external.txt",
			linkTarget: filepath.Join(t.TempDir(), "outside.txt"),
		},
		{
			name:       "relative target escaping root",
			linkName:   "escape.txt",
			linkTarget: filepath.Join("..", "outside.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			src := filepath.Join(root, "src")
			dst := filepath.Join(root, "dst")
			require.NoError(t, os.MkdirAll(src, 0o755))

			outside := filepath.Join(root, "outside.txt")
			require.NoError(t, os.WriteFile(outside, []byte("outside"), 0o644))
			require.NoError(t, os.Symlink(tt.linkTarget, filepath.Join(src, tt.linkName)))

			err := CopyDir(src, dst)
			require.Error(t, err)
			assert.ErrorContains(t, err, "points outside source tree")
		})
	}
}
