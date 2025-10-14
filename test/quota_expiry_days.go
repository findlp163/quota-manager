package main

import (
	"fmt"
	"time"

	"quota-manager/internal/models"
)

// testExpiryDaysQuotaExpiry tests ExpiryDays quota expiry scenarios
// Includes three test scenarios:
// 1. Single quota usage after expiry
// 2. Used quota less than expired quota
// 3. Used quota greater than expired quota
func testExpiryDaysQuotaExpiry(ctx *TestContext) TestResult {
	startTime := time.Now()
	fmt.Println("Starting test: ExpiryDays - Quota Expiry Scenarios")

	// Create test users
	user1 := createTestUser("test_user_single_expire", "Test User Single Expire", 0)
	user2 := createTestUser("test_user_lt_expired", "Test User LT Expired", 0)
	user3 := createTestUser("test_user_gt_expired", "Test User GT Expired", 0)

	// Batch create users
	if err := ctx.DB.AuthDB.Create(&[]*models.UserInfo{user1, user2, user3}).Error; err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create test users failed: %v", err)}
	}

	// Scenario 1: Single quota usage after expiry
	// User1: Quota 100, Used 30, Quota expired
	quota1, err := createExpiredTestQuota(ctx, user1.ID, 100.0)
	if err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create user1 expired quota failed: %v", err)}
	}
	ctx.MockQuotaStore.SetQuota(user1.ID, 100.0)
	ctx.MockQuotaStore.SetUsed(user1.ID, 30.0)

	// Scenario 2: Used quota < Expired quota (30 < 100)
	// User2: Quota 100(expired) + Quota 80(valid), Used 30, Expected used quota 0, remaining quota 80
	expiredQuota2, err := createExpiredTestQuota(ctx, user2.ID, 110.0)
	if err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create user2 expired quota failed: %v", err)}
	}
	validQuota2, err := createValidTestQuota(ctx, user2.ID, 80.0)
	if err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create user2 valid quota failed: %v", err)}
	}
	ctx.MockQuotaStore.SetQuota(user2.ID, 180.0)
	ctx.MockQuotaStore.SetUsed(user2.ID, 30.0)

	// Scenario 3: Used quota > Expired quota (120 > 50)
	// User3: Quota 50(expired) + Quota 100(valid), Used 120, Expected used quota 70, remaining quota 30
	expiredQuota3, err := createExpiredTestQuota(ctx, user3.ID, 10.0)
	if err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create user3 expired quota failed: %v", err)}
	}
	validQuota3, err := createValidTestQuota(ctx, user3.ID, 100.0)
	if err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create user3 valid quota failed: %v", err)}
	}
	ctx.MockQuotaStore.SetQuota(user3.ID, 110.0)
	ctx.MockQuotaStore.SetUsed(user3.ID, 40.0)

	// Execute quota expiry task
	if err := executeExpireQuotasTask(ctx); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Execute quota expiry task failed: %v", err)}
	}

	// Verify Scenario 1: Single quota expiry
	if err := verifyQuotaStatus(ctx, fmt.Sprintf("%d", quota1.ID), models.StatusExpired); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User1 quota status verification failed: %v", err)}
	}
	if err := verifyUserValidQuotaCount(ctx, user1.ID, 0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User1 valid quota count verification failed: %v", err)}
	}
	if err := verifyUserExpiredQuotaCount(ctx, user1.ID, 1); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User1 expired quota count verification failed: %v", err)}
	}
	if err := verifyMockQuotaStoreTotalQuota(ctx, user1.ID, 0.0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User1 total quota sync verification failed: %v", err)}
	}
	if err := verifyMockQuotaStoreUsedQuota(ctx, user1.ID, 0.0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User1 used quota sync verification failed: %v", err)}
	}

	// Verify Scenario 2: Used quota < Expired quota
	if err := verifyQuotaStatus(ctx, fmt.Sprintf("%d", expiredQuota2.ID), models.StatusExpired); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User2 expired quota status verification failed: %v", err)}
	}
	if err := verifyQuotaStatus(ctx, fmt.Sprintf("%d", validQuota2.ID), models.StatusValid); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User2 valid quota status verification failed: %v", err)}
	}
	if err := verifyUserValidQuotaCount(ctx, user2.ID, 1); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User2 valid quota count verification failed: %v", err)}
	}
	if err := verifyUserExpiredQuotaCount(ctx, user2.ID, 1); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User2 expired quota count verification failed: %v", err)}
	}
	if err := verifyMockQuotaStoreTotalQuota(ctx, user2.ID, 80.0); err != nil { // Total quota: 180 - 100(expired) = 80
		return TestResult{Passed: false, Message: fmt.Sprintf("User2 total quota sync verification failed: %v", err)}
	}
	if err := verifyMockQuotaStoreUsedQuota(ctx, user2.ID, 0.0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User2 used quota sync verification failed: %v", err)}
	}

	// Verify Scenario 3: Used quota > Expired quota
	if err := verifyQuotaStatus(ctx, fmt.Sprintf("%d", expiredQuota3.ID), models.StatusExpired); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User3 expired quota status verification failed: %v", err)}
	}
	if err := verifyQuotaStatus(ctx, fmt.Sprintf("%d", validQuota3.ID), models.StatusValid); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User3 valid quota status verification failed: %v", err)}
	}
	if err := verifyUserValidQuotaCount(ctx, user3.ID, 1); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User3 valid quota count verification failed: %v", err)}
	}
	if err := verifyUserExpiredQuotaCount(ctx, user3.ID, 1); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User3 expired quota count verification failed: %v", err)}
	}
	if err := verifyMockQuotaStoreTotalQuota(ctx, user3.ID, 100.0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User3 total quota sync verification failed: %v", err)}
	}
	if err := verifyMockQuotaStoreUsedQuota(ctx, user3.ID, 30.0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User3 used quota sync verification failed: %v", err)}
	}

	// Verify audit records
	if err := verifyQuotaExpiryAuditExists(ctx, user1.ID, -100.0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User1 audit record verification failed: %v", err)}
	}
	if err := verifyQuotaExpiryAuditExists(ctx, user2.ID, -110.0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User2 audit record verification failed: %v", err)}
	}
	if err := verifyQuotaExpiryAuditExists(ctx, user3.ID, -10.0); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("User3 audit record verification failed: %v", err)}
	}

	duration := time.Since(startTime)
	return TestResult{
		Passed:    true,
		Message:   "ExpiryDays quota expiry test succeeded: Verified single quota expiry, used quota less than and greater than expired quota, quota not expired scenarios",
		Duration:  duration,
		TestName:  "testExpiryDaysQuotaExpiry",
		StartTime: startTime,
		EndTime:   time.Now(),
	}
}
