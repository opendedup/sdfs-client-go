package test

import (
	"context"

	"testing"

	"github.com/opendedup/sdfs-client-go/api"
	"github.com/stretchr/testify/assert"
)

var testUser = "test"
var testPassword = "iL0ve#ranges"
var testPasswordBad = "password"
var testDescription = "bladebla"

func TestUserLifeCycle(t *testing.T) {
	connection := connect(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer connection.CloseConnection(ctx)
	defer cancel()
	permissions := []string{"CONFIG_READ"}
	err := connection.AddUser(ctx, testUser, testPasswordBad, testDescription, permissions)
	assert.NotNil(t, err)
	err = connection.AddUser(ctx, testUser, testPassword, testDescription, permissions)
	assert.Nil(t, err)
	users, err := connection.ListUsers(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	user := users[0]
	assert.Equal(t, true, user.Permissions.CONFIG_READ)
	assert.Equal(t, testDescription, user.Description)
	assert.Equal(t, testUser, user.User)
	permissions = []string{"METADATA_READ", "METADATA_WRITE", "FILE_READ", "FILE_WRITE", "FILE_DELETE", "VOLUME_READ", "CONFIG_READ",
		"CONFIG_WRITE", "EVENT_READ", "AUTH_READ", "AUTH_WRITE", "ADMIN"}
	err = connection.SetSdfsPermissions(ctx, testUser, permissions)
	assert.Nil(t, err)
	users, err = connection.ListUsers(ctx)
	assert.Nil(t, err)
	user = users[0]
	assert.Equal(t, true, user.Permissions.METADATA_READ)
	assert.Equal(t, true, user.Permissions.METADATA_WRITE)
	assert.Equal(t, true, user.Permissions.FILE_READ)
	assert.Equal(t, true, user.Permissions.FILE_WRITE)
	assert.Equal(t, true, user.Permissions.FILE_DELETE)
	assert.Equal(t, true, user.Permissions.VOLUME_READ)
	assert.Equal(t, true, user.Permissions.CONFIG_READ)
	assert.Equal(t, true, user.Permissions.CONFIG_WRITE)
	assert.Equal(t, true, user.Permissions.EVENT_READ)
	assert.Equal(t, true, user.Permissions.AUTH_READ)
	assert.Equal(t, true, user.Permissions.AUTH_WRITE)
	assert.Equal(t, true, user.Permissions.ADMIN)
	err = connection.DeleteUser(ctx, testUser)
	assert.Nil(t, err)
	users, err = connection.ListUsers(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(users))
}

func TestUserAuthenitcation(t *testing.T) {
	connection := connect(t, false)
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	permissions := []string{"CONFIG_READ"}
	err := connection.AddUser(ctx, testUser, testPassword, testDescription, permissions)

	assert.Nil(t, err)
	err = connection.CloseConnection(ctx)
	assert.Nil(t, err)
	api.UserName = testUser
	api.Password = testPassword
	connection = connect(t, false)
	_, err = connection.DSEInfo(ctx)
	assert.Nil(t, err)
	connection.CloseConnection(ctx)
	api.Password = testPasswordBad
	connection = connect(t, false)
	_, err = connection.DSEInfo(ctx)
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
	api.UserName = "admin"
	api.Password = password
	connection = connect(t, false)
	connection.SetSdfsPassword(ctx, testUser, testPassword+"111")
	connection.CloseConnection(ctx)
	api.UserName = testUser
	api.Password = testPassword
	_, err = connection.DSEInfo(ctx)
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
	api.UserName = testUser
	api.Password = testPassword + "111"
	_, err = connection.DSEInfo(ctx)
	assert.Nil(t, err)
	connection.CloseConnection(ctx)
	api.UserName = "admin"
	api.Password = password
	connection = connect(t, false)
	err = connection.DeleteUser(ctx, testUser)
	assert.Nil(t, err)
	users, err := connection.ListUsers(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(users))
	connection.CloseConnection(ctx)
}

func TestUserFileAuthorization(t *testing.T) {
	connection := connect(t, false)
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	permissions := []string{"CONFIG_READ"}
	err := connection.AddUser(ctx, testUser, testPassword, testDescription, permissions)

	assert.Nil(t, err)
	err = connection.CloseConnection(ctx)
	assert.Nil(t, err)
	api.UserName = testUser
	api.Password = testPassword
	connection = connect(t, false)
	_, _, err = makeGenericFileNt(ctx, connection, "", 1024)
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
	api.UserName = "admin"
	api.Password = password
	connection = connect(t, false)
	permissions = []string{"FILE_WRITE", "FILE_READ", "METADATA_READ"}
	err = connection.SetSdfsPermissions(ctx, testUser, permissions)
	assert.Nil(t, err)
	err = connection.CloseConnection(ctx)
	assert.Nil(t, err)
	api.UserName = testUser
	api.Password = testPassword
	connection = connect(t, false)
	fn, _, err := makeGenericFile(ctx, t, connection, "", 1024)
	assert.Nil(t, err)
	err = connection.DeleteFile(ctx, fn)
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
	api.UserName = "admin"
	api.Password = password
	connection = connect(t, false)
	permissions = []string{"FILE_DELETE"}
	err = connection.SetSdfsPermissions(ctx, testUser, permissions)
	assert.Nil(t, err)
	err = connection.CloseConnection(ctx)
	assert.Nil(t, err)
	api.UserName = testUser
	api.Password = testPassword
	connection = connect(t, false)
	err = connection.DeleteFile(ctx, fn)
	assert.Nil(t, err)
	err = connection.AddUser(ctx, "testUser1", testPassword, testDescription, permissions)
	assert.NotNil(t, err)
	permissions = []string{"FILE_DELETE"}
	err = connection.SetSdfsPermissions(ctx, testUser, permissions)
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
	api.UserName = "admin"
	api.Password = password
	connection = connect(t, false)
	defer connection.CloseConnection(ctx)
	err = connection.DeleteUser(ctx, testUser)
	assert.Nil(t, err)
	users, err := connection.ListUsers(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(users))
}
