package main

import "testing"

func TestCheckAuthVersion_MatchingVersion_Allowed(t *testing.T) {
	if !checkAuthVersionAllowed(1, 1) {
		t.Error("matching versions should be allowed")
	}
}

func TestCheckAuthVersion_StaleJWT_Rejected(t *testing.T) {
	if checkAuthVersionAllowed(1, 2) {
		t.Error("stale JWT version should be rejected when DB version is higher")
	}
}

func TestCheckAuthVersion_FutureJWT_Rejected(t *testing.T) {
	if checkAuthVersionAllowed(3, 2) {
		t.Error("JWT version ahead of DB should be rejected")
	}
}

func TestCheckAuthVersion_MatchingHighVersion_Allowed(t *testing.T) {
	if !checkAuthVersionAllowed(5, 5) {
		t.Error("matching high versions should be allowed")
	}
}

func TestCheckAuthVersion_LegacyToken_DBVersion1_Allowed(t *testing.T) {
	// Legacy token: AuthVersion == 0 (field absent in JSON), treated as 1.
	// DB version is still 1 → allowed.
	if !checkAuthVersionAllowed(0, 1) {
		t.Error("legacy token (version 0) against DB version 1 should be allowed")
	}
}

func TestCheckAuthVersion_LegacyToken_DBVersion2_Rejected(t *testing.T) {
	// Legacy token against a DB that has already had sessions invalidated → reject.
	if checkAuthVersionAllowed(0, 2) {
		t.Error("legacy token (version 0) against DB version 2 should be rejected")
	}
}

// TestPasswordReset_OldTokenRejected documents that a token issued before an
// admin password reset is permanently rejected after the reset.
//
// ResetPasswordAndInvalidateSessions increments auth_version in the same
// transaction as the password update, so there is no window between the password
// change and session invalidation.
func TestPasswordReset_OldTokenRejected(t *testing.T) {
	const (
		dbVersionBefore = 1
		dbVersionAfter  = 2 // incremented by ResetPasswordAndInvalidateSessions
		oldTokenVersion = 1
		newTokenVersion = 2
	)

	// Pre-condition: old token was valid before reset.
	if !checkAuthVersionAllowed(oldTokenVersion, dbVersionBefore) {
		t.Fatal("precondition: old token must be valid before password reset")
	}

	// Immediately after password reset, old token is rejected.
	if checkAuthVersionAllowed(oldTokenVersion, dbVersionAfter) {
		t.Error("old token must be rejected after admin password reset")
	}

	// New login after reset yields auth_version = 2 → accepted.
	if !checkAuthVersionAllowed(newTokenVersion, dbVersionAfter) {
		t.Error("new token issued after password reset must be accepted")
	}
}

// TestRefresh_NewTokenCarriesUpdatedAuthVersion documents that a token obtained
// after a refresh carries the latest auth_version from the DB, so a session
// that was valid at login continues to work after the access token expires and
// the refresh endpoint issues a new one.
func TestRefresh_NewTokenCarriesUpdatedAuthVersion(t *testing.T) {
	const (
		dbVersion        = 3
		freshTokenVersion = 3 // matches the DB after a refresh
	)

	if !checkAuthVersionAllowed(freshTokenVersion, dbVersion) {
		t.Error("token issued by refresh endpoint (carrying current auth_version) must be accepted")
	}
}

// TestConcurrentExpiry_OnlyFirstRefreshSucceeds documents the expectation for
// concurrent requests whose access token expired at the same time. The first
// request refreshes and obtains a new token (auth_version still matches). Any
// subsequent retries using that new token must succeed. Stale tokens that
// carry an old auth_version after the user invalidated sessions are still
// correctly rejected regardless of concurrency.
func TestConcurrentExpiry_StaleVersionStillRejected(t *testing.T) {
	const (
		dbVersionAfterInvalidation = 4
		oldTokenVersion            = 3
		newTokenVersion            = 4
	)

	if checkAuthVersionAllowed(oldTokenVersion, dbVersionAfterInvalidation) {
		t.Error("stale token from before session invalidation must be rejected even under concurrent load")
	}
	if !checkAuthVersionAllowed(newTokenVersion, dbVersionAfterInvalidation) {
		t.Error("token from new login after invalidation must be accepted")
	}
}

// TestRoleChange_OldTokenRejected documents that a role change paired with
// session invalidation (auth_version increment) causes old tokens to be
// rejected on the next request, forcing the user to log in again and receive
// a token with the updated role and auth_version.
func TestRoleChange_OldTokenRejected(t *testing.T) {
	const (
		dbVersionBefore = 2
		dbVersionAfter  = 3 // incremented when role-change triggers InvalidateSessions
		tokenVersion    = 2
	)

	if !checkAuthVersionAllowed(tokenVersion, dbVersionBefore) {
		t.Fatal("precondition: token must be valid before role change")
	}
	if checkAuthVersionAllowed(tokenVersion, dbVersionAfter) {
		t.Error("token issued before role change must be rejected after session invalidation")
	}
}

// TestDeactivateReactivate_OldTokenAlwaysRejected documents the full lifecycle
// property: a token issued before deactivation must be rejected both immediately
// after deactivation AND after subsequent reactivation.
//
// DB state transitions:
//
//	initial:      auth_version = 1
//	deactivate:   auth_version = 2  (employees.active=false, users.active=false, tokens revoked)
//	reactivate:   auth_version = 2  (employees.active=true,  users.active=true,  unchanged)
//
// The pre-deactivation token carries auth_version = 1.
func TestDeactivateReactivate_OldTokenAlwaysRejected(t *testing.T) {
	const (
		dbVersionInitial         = 1
		dbVersionAfterDeactivate = 2 // incremented by DeactivateEmployeeWithInvalidation
		dbVersionAfterReactivate = 2 // ActivateEmployeeAccount does NOT change auth_version
		oldTokenVersion          = 1 // token issued at initial state
		newTokenVersion          = 2 // token issued after reactivation (fresh login)
	)

	// Pre-condition: old token was valid before deactivation.
	if !checkAuthVersionAllowed(oldTokenVersion, dbVersionInitial) {
		t.Fatal("precondition: old token must be valid before deactivation")
	}

	// Immediately after deactivation.
	if checkAuthVersionAllowed(oldTokenVersion, dbVersionAfterDeactivate) {
		t.Error("old token must be rejected immediately after deactivation")
	}

	// After reactivation — auth_version is still 2, so old token with version 1 stays rejected.
	if checkAuthVersionAllowed(oldTokenVersion, dbVersionAfterReactivate) {
		t.Error("old token must remain rejected even after employee is reactivated")
	}

	// New login after reactivation yields auth_version = 2 → accepted.
	if !checkAuthVersionAllowed(newTokenVersion, dbVersionAfterReactivate) {
		t.Error("new token issued after reactivation must be accepted")
	}
}
