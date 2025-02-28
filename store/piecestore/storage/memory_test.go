package storage

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupMemoryTest(t *testing.T) *memoryStore {
	return &memoryStore{name: mockBucket}
}

func TestMemory_String(t *testing.T) {
	store := setupMemoryTest(t)
	result := store.String()
	assert.Equal(t, "memory://mockBucket/", result)
}

func TestMemory_GetSuccess(t *testing.T) {
	cases := []struct {
		name         string
		key          string
		wantedResult string
		wantedErr    error
	}{
		{
			name:         "memory_get_success_test1",
			key:          mockKey,
			wantedResult: mockAccessKey,
			wantedErr:    nil,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryTest(t)
			store.objects = map[string]*memoryObject{
				mockKey: {data: []byte(mockAccessKey)},
			}
			data, err := store.GetObject(context.TODO(), tt.key, 0, 0)
			assert.Equal(t, tt.wantedErr, err)
			data1, err := io.ReadAll(data)
			if err != nil {
				t.Fatalf("Get io.ReadAll error: %s", err)
			}
			assert.Equal(t, tt.wantedResult, string(data1))
		})
	}
}

func TestMemory_GetError(t *testing.T) {
	cases := []struct {
		name      string
		key       string
		wantedErr error
	}{
		{
			name:      "memory_get_error_test1",
			key:       emptyString,
			wantedErr: ErrInvalidObjectKey,
		},
		{
			name:      "memory_get_error_test2",
			key:       mockKey,
			wantedErr: ErrNoSuchObject,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryTest(t)
			store.objects = map[string]*memoryObject{
				mockSecretKey: {data: []byte(mockSecretKey)},
			}
			data, err := store.GetObject(context.TODO(), tt.key, 0, 0)
			assert.Equal(t, tt.wantedErr, err)
			assert.Equal(t, nil, data)
		})
	}
}

func TestMemory_Put(t *testing.T) {
	cases := []struct {
		name      string
		key       string
		data      string
		wantedErr error
	}{
		{
			name:      "memory_put_test1",
			key:       emptyString,
			data:      mockEndpoint,
			wantedErr: ErrInvalidObjectKey,
		},
		{
			name:      "memory_put_test2",
			key:       mockAccessKey,
			data:      mockEndpoint,
			wantedErr: nil,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryTest(t)
			store.objects = map[string]*memoryObject{
				mockAccessKey: {data: []byte(mockSecretKey)},
			}
			err := store.PutObject(context.TODO(), tt.key, strings.NewReader(tt.data))
			assert.Equal(t, tt.wantedErr, err)
		})
	}
}

func TestMemory_Delete(t *testing.T) {
	cases := []struct {
		name      string
		key       string
		wantedErr error
	}{
		{
			name:      "memory_delete_test1",
			key:       mockKey,
			wantedErr: nil,
		},
		{
			name:      "memory_delete_test2",
			key:       mockAccessKey,
			wantedErr: nil,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryTest(t)
			store.objects = map[string]*memoryObject{
				mockAccessKey: {data: []byte(mockSecretKey)},
			}
			err := store.DeleteObject(context.TODO(), tt.key)
			assert.Equal(t, tt.wantedErr, err)
		})
	}
}

func TestMemory_HeadSuccess(t *testing.T) {
	cases := []struct {
		name      string
		key       string
		wantedErr error
	}{
		{
			name:      "memory_head_success_test1",
			key:       mockAccessKey,
			wantedErr: nil,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryTest(t)
			store.objects = map[string]*memoryObject{
				mockAccessKey: {data: []byte(mockSecretKey)},
			}
			obj, err := store.HeadObject(context.TODO(), tt.key)
			assert.Equal(t, tt.wantedErr, err)
			assert.Equal(t, mockAccessKey, obj.Key())
		})
	}
}

func TestMemory_HeadError(t *testing.T) {
	cases := []struct {
		name      string
		key       string
		wantedErr error
	}{
		{
			name:      "memory_head_error_test1",
			key:       emptyString,
			wantedErr: ErrInvalidObjectKey,
		},
		{
			name:      "memory_head_error_test2",
			key:       mockKey,
			wantedErr: os.ErrNotExist,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryTest(t)
			obj, err := store.HeadObject(context.TODO(), tt.key)
			assert.Equal(t, tt.wantedErr, err)
			assert.Equal(t, nil, obj)
		})
	}
}

func TestMemory_ListSuccess(t *testing.T) {
	cases := []struct {
		name      string
		prefix    string
		wantedErr error
	}{
		{
			name:      "memory_list_success_test1",
			prefix:    mockAccessKey,
			wantedErr: nil,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryTest(t)
			store.objects = map[string]*memoryObject{
				mockAccessKey: {data: []byte(mockSecretKey)},
			}
			objs, err := store.ListObjects(context.TODO(), tt.prefix, emptyString, emptyString, 1)
			assert.Equal(t, tt.wantedErr, err)
			assert.Equal(t, mockAccessKey, objs[0].Key())
		})
	}
}

func TestMemory_ListError(t *testing.T) {
	cases := []struct {
		name      string
		delimiter string
		wantedErr error
	}{
		{
			name:      "memory_list_error_test1",
			delimiter: mockKey,
			wantedErr: ErrUnsupportedDelimiter,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryTest(t)
			objs, err := store.ListObjects(context.TODO(), emptyString, emptyString, tt.delimiter, 0)
			assert.Equal(t, tt.wantedErr, err)
			assert.Equal(t, 0, len(objs))
		})
	}
}

func TestMemory_ListAll(t *testing.T) {
	store := setupMemoryTest(t)
	_, err := store.ListAllObjects(context.TODO(), emptyString, emptyString)
	assert.Equal(t, ErrUnsupportedMethod, err)
}
