package logstash

import (
	"testing"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	log := New(10)

	s1 := "[test1 ] TRACE test1"
	s2 := "[test2 ] ERROR test2"
	s3 := "[test1 ] TRACE test3"

	// old to new
	log.Write([]byte(s1))
	log.Write([]byte(s2))
	log.Write([]byte(s3))

	idx := log.data

	assert.Equal(t, []string{s3, s2, s1}, log.All(nil, jww.LevelTrace, 0))
	assert.Equal(t, []string{s3, s2, s1}, log.All([]string{}, jww.LevelTrace, 0))

	assert.Equal(t, []string{s3, s1}, log.All([]string{"test1"}, jww.LevelTrace, 0))
	assert.Equal(t, []string{s3, s2, s1}, log.All(nil, jww.LevelTrace, 0))
	assert.Nil(t, log.All(nil, jww.LevelFatal, 0))

	assert.Equal(t, idx, log.data, "data should not be changed after All() call")
	assert.Equal(t, []string{"test1", "test2"}, log.Areas())
}
