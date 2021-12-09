package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validVersion(t *testing.T) {

	tests := []struct {
		name     string
		actual   *Version
		expected *Version
		val      int
	}{
		{"expect early minor version", MustParse("1.5"), MustParse("1.4"), -1},
		{"expect same version", MustParse("1.5"), MustParse("1.5"), 0},
		{"expect newer version", MustParse("1.5"), MustParse("1.6"), 1},
		{"full semver is not a factor", MustParse("1.5.8"), MustParse("1.5.0"), 0},
		{"expect early major version", MustParse("2.4"), MustParse("1.4"), -1},
		{"expect newer major version", MustParse("1.4"), MustParse("2.4"), 1},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			val := tt.expected.CompareMajorMinor(tt.actual)
			assert.Equal(t, val, tt.val)
		})
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"clean ver", "1.0.0", "1.0.0"},
		{"clean ver", "v1.0.0", "1.0.0"},
		{"short ver", "v1.0", "1.0"},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			result := Clean(tt.actual)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGet(t *testing.T) {
	info := Get()

	assert.NotNil(t, info)
	assert.NotEmpty(t, info.BuildDate)
	assert.NotEmpty(t, info.Compiler)
	assert.NotEmpty(t, info.GitCommit)
	assert.NotEmpty(t, info.GitVersion)
	assert.NotEmpty(t, info.GoVersion)
	assert.NotEmpty(t, info.Platform)

	assert.Equal(t, info.GitVersion, info.String())
}

func TestNew(t *testing.T) {
	v, err := New("3.4.5")

	assert.NoError(t, err)
	assert.Equal(t, uint64(3), v.Major())
	assert.Equal(t, uint64(4), v.Minor())
	assert.Equal(t, uint64(5), v.Patch())
}

func TestNew_BadVersion(t *testing.T) {
	_, err := New("not-a-version")

	assert.Error(t, err)
}

func TestFromGithubVersion(t *testing.T) {
	v, err := FromGithubVersion("v3.4.5")

	assert.NoError(t, err)
	assert.Equal(t, uint64(3), v.Major())
	assert.Equal(t, uint64(4), v.Minor())
	assert.Equal(t, uint64(5), v.Patch())
}
