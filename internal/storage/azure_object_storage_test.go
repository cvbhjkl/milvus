// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureObjectStorage(t *testing.T) {
	ctx := context.Background()
	bucketName := Params.MinioCfg.BucketName.GetValue()
	config := config{
		bucketName:    bucketName,
		createBucket:  true,
		useIAM:        false,
		cloudProvider: "azure",
	}

	t.Run("test initialize", func(t *testing.T) {
		var err error
		config.bucketName = ""
		_, err = newAzureObjectStorageWithConfig(ctx, &config)
		assert.Error(t, err)
		config.bucketName = bucketName
		_, err = newAzureObjectStorageWithConfig(ctx, &config)
		assert.Equal(t, err, nil)
	})

	t.Run("test load", func(t *testing.T) {
		testCM, err := newAzureObjectStorageWithConfig(ctx, &config)
		assert.Equal(t, err, nil)
		defer testCM.DeleteContainer(ctx, config.bucketName, &azblob.DeleteContainerOptions{})

		prepareTests := []struct {
			key   string
			value []byte
		}{
			{"abc", []byte("123")},
			{"abcd", []byte("1234")},
			{"key_1", []byte("111")},
			{"key_2", []byte("222")},
			{"key_3", []byte("333")},
		}

		for _, test := range prepareTests {
			err := testCM.PutObject(ctx, config.bucketName, test.key, bytes.NewReader(test.value), int64(len(test.value)))
			require.NoError(t, err)
		}

		loadTests := []struct {
			isvalid       bool
			loadKey       string
			expectedValue []byte

			description string
		}{
			{true, "abc", []byte("123"), "load valid key abc"},
			{true, "abcd", []byte("1234"), "load valid key abcd"},
			{true, "key_1", []byte("111"), "load valid key key_1"},
			{true, "key_2", []byte("222"), "load valid key key_2"},
			{true, "key_3", []byte("333"), "load valid key key_3"},
			{false, "key_not_exist", []byte(""), "load invalid key key_not_exist"},
			{false, "/", []byte(""), "load leading slash"},
		}

		for _, test := range loadTests {
			t.Run(test.description, func(t *testing.T) {
				if test.isvalid {
					got, err := testCM.GetObject(ctx, config.bucketName, test.loadKey, 0, 1024)
					assert.NoError(t, err)
					contentData, err := io.ReadAll(got)
					assert.NoError(t, err)
					assert.Equal(t, len(contentData), len(test.expectedValue))
					assert.Equal(t, test.expectedValue, contentData)
					statSize, err := testCM.StatObject(ctx, config.bucketName, test.loadKey)
					assert.NoError(t, err)
					assert.Equal(t, statSize, int64(len(contentData)))
					_, err = testCM.GetObject(ctx, config.bucketName, test.loadKey, 1, 1023)
					assert.NoError(t, err)
				} else {
					if test.loadKey == "/" {
						got, err := testCM.GetObject(ctx, config.bucketName, test.loadKey, 0, 1024)
						assert.Error(t, err)
						assert.Empty(t, got)
						return
					}
					got, err := testCM.GetObject(ctx, config.bucketName, test.loadKey, 0, 1024)
					assert.Error(t, err)
					assert.Empty(t, got)
				}
			})
		}

		loadWithPrefixTests := []struct {
			isvalid       bool
			prefix        string
			expectedValue [][]byte

			description string
		}{
			{true, "abc", [][]byte{[]byte("123"), []byte("1234")}, "load with valid prefix abc"},
			{true, "key_", [][]byte{[]byte("111"), []byte("222"), []byte("333")}, "load with valid prefix key_"},
			{true, "prefix", [][]byte{}, "load with valid but not exist prefix prefix"},
		}

		for _, test := range loadWithPrefixTests {
			t.Run(test.description, func(t *testing.T) {
				gotk, _, err := testCM.ListObjects(ctx, config.bucketName, test.prefix, false)
				assert.NoError(t, err)
				assert.Equal(t, len(test.expectedValue), len(gotk))
				for _, key := range gotk {
					err := testCM.RemoveObject(ctx, config.bucketName, key)
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("test list", func(t *testing.T) {
		testCM, err := newAzureObjectStorageWithConfig(ctx, &config)
		assert.Equal(t, err, nil)
		defer testCM.DeleteContainer(ctx, config.bucketName, &azblob.DeleteContainerOptions{})

		prepareTests := []struct {
			valid bool
			key   string
			value []byte
		}{
			{false, "abc/", []byte("123")},
			{true, "abc/d", []byte("1234")},
			{false, "abc/d/e", []byte("12345")},
			{true, "abc/e/d", []byte("12354")},
			{true, "key_/1/1", []byte("111")},
			{true, "key_/1/2", []byte("222")},
			{false, "key_/1/2/3", []byte("333")},
			{true, "key_/2/3", []byte("333")},
		}

		for _, test := range prepareTests {
			err := testCM.PutObject(ctx, config.bucketName, test.key, bytes.NewReader(test.value), int64(len(test.value)))
			require.Nil(t, err)
			if !test.valid {
				err := testCM.RemoveObject(ctx, config.bucketName, test.key)
				require.Nil(t, err)
			}
		}

		insertWithPrefixTests := []struct {
			recursive     bool
			prefix        string
			expectedValue []string
		}{
			{true, "abc/", []string{"abc/d", "abc/e/d"}},
			{true, "key_/", []string{"key_/1/1", "key_/1/2", "key_/2/3"}},
			{false, "abc/", []string{"abc/d", "abc/e/"}},
			{false, "key_/", []string{"key_/1/", "key_/2/"}},
		}

		for _, test := range insertWithPrefixTests {
			t.Run(fmt.Sprintf("prefix: %s, recursive: %t", test.prefix, test.recursive), func(t *testing.T) {
				gotk, _, err := testCM.ListObjects(ctx, config.bucketName, test.prefix, test.recursive)
				assert.NoError(t, err)
				assert.Equal(t, len(test.expectedValue), len(gotk))
				for _, key := range gotk {
					assert.Contains(t, test.expectedValue, key)
				}
			})
		}
	})

	t.Run("test useIAM", func(t *testing.T) {
		var err error
		config.useIAM = true
		_, err = newAzureObjectStorageWithConfig(ctx, &config)
		assert.Error(t, err)
		os.Setenv("AZURE_CLIENT_ID", "00000000-0000-0000-0000-00000000000")
		os.Setenv("AZURE_TENANT_ID", "00000000-0000-0000-0000-00000000000")
		os.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/var/run/secrets/tokens/azure-identity-token")
		_, err = newAzureObjectStorageWithConfig(ctx, &config)
		assert.Error(t, err)
		config.useIAM = false
	})

	t.Run("test key secret", func(t *testing.T) {
		var err error
		connectionString := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
		os.Setenv("AZURE_STORAGE_CONNECTION_STRING", "")
		config.accessKeyID = "devstoreaccount1"
		config.secretAccessKeyID = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
		config.address = "core.windows.net"
		_, err = newAzureObjectStorageWithConfig(ctx, &config)
		assert.Error(t, err)
		os.Setenv("AZURE_STORAGE_CONNECTION_STRING", connectionString)
	})
}
