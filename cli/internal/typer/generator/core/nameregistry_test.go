package core_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/stretchr/testify/assert"
)

func TestNewNameRegistry(t *testing.T) {
	handler := core.DefaultCollisionHandler
	registry := core.NewNameRegistry(handler)

	assert.NotNil(t, registry)
}

func TestRegisterName_BasicRegistration(t *testing.T) {
	registry := core.NewNameRegistry(core.DefaultCollisionHandler)

	name, err := registry.RegisterName("user_id", "types", "UserId")

	assert.NoError(t, err)
	assert.Equal(t, "UserId", name)
}

func TestRegisterName_SameIdReturnsExistingName(t *testing.T) {
	registry := core.NewNameRegistry(core.DefaultCollisionHandler)

	// Register the same id twice
	name1, err1 := registry.RegisterName("user_id", "types", "UserId")
	name2, err2 := registry.RegisterName("user_id", "types", "DifferentName")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, "UserId", name1)
	assert.Equal(t, "UserId", name2) // Should return the originally registered name
}

func TestRegisterName_CollisionHandling(t *testing.T) {
	registry := core.NewNameRegistry(core.DefaultCollisionHandler)

	// Register first name
	name1, err1 := registry.RegisterName("id1", "types", "UserId")
	assert.NoError(t, err1)
	assert.Equal(t, "UserId", name1)

	// Register different id with same name - should trigger collision handler
	name2, err2 := registry.RegisterName("id2", "types", "UserId")
	assert.NoError(t, err2)
	assert.Equal(t, "UserId1", name2)

	// Register another collision
	name3, err3 := registry.RegisterName("id3", "types", "UserId")
	assert.NoError(t, err3)
	assert.Equal(t, "UserId2", name3)
}

func TestRegisterName_DifferentScopes(t *testing.T) {
	registry := core.NewNameRegistry(core.DefaultCollisionHandler)

	// Register same name in different scopes - should not conflict
	name1, err1 := registry.RegisterName("id1", "types", "UserId")
	name2, err2 := registry.RegisterName("id2", "methods", "UserId")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, "UserId", name1)
	assert.Equal(t, "UserId", name2)
}

func TestRegisterName_SameScopeDifferentIds(t *testing.T) {
	registry := core.NewNameRegistry(core.DefaultCollisionHandler)

	// Register different names in same scope - should not conflict
	name1, err1 := registry.RegisterName("id1", "types", "UserId")
	name2, err2 := registry.RegisterName("id2", "types", "EmailAddress")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, "UserId", name1)
	assert.Equal(t, "EmailAddress", name2)
}

func TestRegisterName_CustomCollisionHandler(t *testing.T) {
	customHandler := func(name string, existingNames []string) string {
		return "Custom_" + name
	}
	registry := core.NewNameRegistry(customHandler)

	// Register first name
	name1, err1 := registry.RegisterName("id1", "types", "UserId")
	assert.NoError(t, err1)
	assert.Equal(t, "UserId", name1)

	// Register collision - should use custom handler
	name2, err2 := registry.RegisterName("id2", "types", "UserId")
	assert.NoError(t, err2)
	assert.Equal(t, "Custom_UserId", name2)
}

func TestRegisterName_EmptyInputs(t *testing.T) {
	registry := core.NewNameRegistry(core.DefaultCollisionHandler)

	// Test empty id
	_, err1 := registry.RegisterName("", "types", "UserId")
	assert.Error(t, err1)

	// Test empty scope
	_, err2 := registry.RegisterName("id1", "", "UserId")
	assert.Error(t, err2)

	// Test empty name
	_, err3 := registry.RegisterName("id1", "types", "")
	assert.Error(t, err3)
}

func TestDefaultCollisionHandler(t *testing.T) {
	existingNames := []string{"UserId", "UserId1", "UserId3"}

	// Should find the first available number (UserId2 is available)
	result := core.DefaultCollisionHandler("UserId", existingNames)
	assert.Equal(t, "UserId2", result)
}

func TestDefaultCollisionHandler_EmptyExistingNames(t *testing.T) {
	existingNames := []string{}

	// Should return the name with "1" suffix since it's a collision scenario
	result := core.DefaultCollisionHandler("UserId", existingNames)
	assert.Equal(t, "UserId1", result)
}

func TestDefaultCollisionHandler_SequentialNumbers(t *testing.T) {
	existingNames := []string{"Name1", "Name2", "Name3"}

	// Should find the next available number after the existing ones
	result := core.DefaultCollisionHandler("Name", existingNames)
	assert.Equal(t, "Name4", result) // Should skip the existing 1, 2, 3 and use 4
}

func TestDefaultCollisionHandler_NonSequentialGap(t *testing.T) {
	existingNames := []string{"Test1", "Test3", "Test5"}

	// Should find the first gap in the sequence
	result := core.DefaultCollisionHandler("Test", existingNames)
	assert.Equal(t, "Test2", result) // Should fill the gap at 2
}

func TestRegisterName_ComplexScenario(t *testing.T) {
	registry := core.NewNameRegistry(core.DefaultCollisionHandler)

	// Register various names across different scopes
	name1, _ := registry.RegisterName("user.id", "types", "UserId")
	name2, _ := registry.RegisterName("user.email", "types", "EmailAddress")
	name3, _ := registry.RegisterName("order.id", "types", "UserId") // Collision
	name4, _ := registry.RegisterName("track.user", "methods", "TrackUser")
	name5, _ := registry.RegisterName("identify.user", "methods", "TrackUser") // Collision in different scope
	name6, _ := registry.RegisterName("user.id", "types", "DifferentName")     // Same id, should return original

	names := []string{name1, name2, name3, name4, name5, name6}

	expected := []string{
		"UserId",       // user.id -> UserId
		"EmailAddress", // user.email -> EmailAddress
		"UserId1",      // order.id -> UserId1 (collision with first)
		"TrackUser",    // track.user -> TrackUser
		"TrackUser1",   // identify.user -> TrackUser1 (collision in methods scope)
		"UserId",       // user.id -> UserId (same id returns original)
	}

	assert.Equal(t, expected, names)
}
