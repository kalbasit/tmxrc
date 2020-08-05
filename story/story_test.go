package story

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	t.Run("no name", func(t *testing.T) {
		assert.EqualError(t, Create("", ""), ErrNameRequired.Error())
	})

	t.Run("file already exists", func(t *testing.T) {
		// create a temporary directory
		dir, err := ioutil.TempDir("", "swm-test-*")
		require.NoError(t, err)
		// delete it once we are done here
		defer func() { os.RemoveAll(dir) }()

		s, err := newStory(t.Name(), "")
		require.NoError(t, err)

		xdg.DataHome = dir
		defer xdg.Reload()

		require.NoError(t, os.MkdirAll(path.Dir(s.filePath()), 0777))
		require.NoError(t, ioutil.WriteFile(s.filePath(), []byte("whatever..."), 0666))

		assert.EqualError(t, Create(t.Name(), ""), ErrStoryExists.Error())
	})

	t.Run("file does not exist", func(t *testing.T) {
		// create a temporary directory
		dir, err := ioutil.TempDir("", "swm-test-*")
		require.NoError(t, err)
		// delete it once we are done here
		defer func() { os.RemoveAll(dir) }()

		s, err := newStory(t.Name(), "")
		require.NoError(t, err)

		xdg.DataHome = dir
		defer xdg.Reload()

		if assert.NoError(t, Create(t.Name(), "")) && assert.FileExists(t, s.filePath()) {
			jb, err := json.Marshal(s)
			require.NoError(t, err, "error compiling the expected json")

			fc, err := ioutil.ReadFile(s.filePath())
			require.NoError(t, err, "error reading the written story file")

			assert.EqualValues(t, bytes.TrimSpace(jb), bytes.TrimSpace(fc))
		}
	})
}