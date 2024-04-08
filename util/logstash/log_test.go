package logstash

import (
	"testing"

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

	assert.Equal(t, []string{s3, s2, s1}, log.All(nil, nil, 0))
	assert.Equal(t, []string{s3, s2, s1}, log.All([]string{}, []string{}, 0))

	assert.Equal(t, []string{s3, s1}, log.All([]string{"test1"}, nil, 0))
	assert.Equal(t, []string{s3, s1}, log.All(nil, []string{"TRACE"}, 0))
	assert.Nil(t, log.All(nil, []string{"FATAL"}, 0))
	assert.Equal(t, idx, log.data, "data should not be changed after All() call")
	assert.Equal(t, []string{"test1", "test2"}, log.Areas())
}
