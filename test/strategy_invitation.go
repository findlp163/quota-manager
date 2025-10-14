package main

import (
	"fmt"

	"quota-manager/internal/models"
)

// testStrategyInviteRegister test strategy invite register
func testStrategyInviteRegister(ctx *TestContext) TestResult {
	// Create test inviter user
	inviterUser := createTestInviterUser("invite_user_register_inviter_test", "Inviter User Test", 0, "")
	if err2 := ctx.DB.AuthDB.Create(inviterUser).Error; err2 != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create user failed: %v", err2)}
	}
	// Create test invited user
	user := createTestInviterUser("invite_user_register_invited_test", "Invited User Test", 0, inviterUser.ID)
	if err := ctx.DB.AuthDB.Create(user).Error; err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create user failed: %v", err)}
	}

	// Debug: verify if inviter ID is correctly set
	if user.InviterID == "" {
		return TestResult{Passed: false, Message: "InviterID is empty after user creation"}
	}
	if user.InviterID != inviterUser.ID {
		return TestResult{Passed: false, Message: fmt.Sprintf("InviterID mismatch: expected %s, got %s", inviterUser.ID, user.InviterID)}
	}

	// Debug: verify user data in database
	var dbUser models.UserInfo
	if err := ctx.DB.AuthDB.Where("id = ?", user.ID).First(&dbUser).Error; err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Failed to query user from database: %v", err)}
	}
	if dbUser.InviterID == "" {
		return TestResult{Passed: false, Message: "InviterID is empty in database"}
	}
	if dbUser.InviterID != inviterUser.ID {
		return TestResult{Passed: false, Message: fmt.Sprintf("Database InviterID mismatch: expected %s, got %s", inviterUser.ID, dbUser.InviterID)}
	}

	// Create single recharge strategy
	strategy := &models.QuotaStrategy{
		Name:   "inviter-register-reward",
		Title:  "Inviter Register Reward",
		Type:   "single",
		Amount: 25,
		Model:  "test-model",
		// Condition: "true()", // Always true condition, all users match
		Condition: `has-inviter()`,
		Status:    true,
	}
	if err := ctx.StrategyService.CreateStrategy(strategy); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create strategy failed: %v", err)}
	}
	// return TestResult{Passed: true, Message: "Single Recharge Strategy Test Succeeded"}

	// user.ID = "635be6cb-3aee-4a56-9c82-abfeaf3283e3"
	// strategy.ID = 1
	// First execute strategy
	users := []models.UserInfo{*user}
	ctx.StrategyService.ExecStrategy(strategy, users)

	// Check first execution result
	var executeCount int64
	ctx.DB.Model(&models.QuotaExecute{}).Where("strategy_id = ? AND user_id = ? AND status = 'completed'", strategy.ID, user.ID).Count(&executeCount)

	if executeCount != 1 {
		return TestResult{Passed: false, Message: fmt.Sprintf("First execution expected 1 time, actually executed %d times", executeCount)}
	}

	// Verify strategy name in audit record (for invitation strategy, check inviter's audit record)
	if err := verifyStrategyNameInAudit(ctx, inviterUser.ID, "inviter-register-reward", models.OperationRecharge); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Strategy name verification failed: %v", err)}
	}

	// Second execute strategy (should be skipped because it has already been executed)
	ctx.StrategyService.ExecStrategy(strategy, users)

	// Check second execution result (should still be 1 time)
	ctx.DB.Model(&models.QuotaExecute{}).Where("strategy_id = ? AND user_id = ? AND status = 'completed'", strategy.ID, user.ID).Count(&executeCount)

	if executeCount != 1 {
		return TestResult{Passed: false, Message: fmt.Sprintf("Single strategy repeated execution, expected still 1 time, actually executed %d times", executeCount)}
	}

	return TestResult{Passed: true, Message: "Invitation Register Reward Strategy Test Succeeded"}
}

// testStrategyInviteStar test strategy invite star reward (reward to inviter when invited user stars)
func testStrategyInviteStar(ctx *TestContext) TestResult {
	// Create test inviter user
	inviterUser := createTestInviterUser("inviter_star_reward_inviter_test", "Invite Star Reward Test", 0, "")
	if err2 := ctx.DB.AuthDB.Create(inviterUser).Error; err2 != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create inviter user failed: %v", err2)}
	}

	// Create test invited user with GitHub star
	user := createTestInviterUser("inviter_star_reward_invited_test", "Invited User For Invite Star Test", 0, inviterUser.ID)
	if err := ctx.DB.AuthDB.Create(user).Error; err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create invited user failed: %v", err)}
	}

	// Create invite star reward strategy (reward goes to inviter)
	strategy := &models.QuotaStrategy{
		Name:      "inviter-star-reward",
		Title:     "Inviter Star Reward",
		Type:      "single",
		Amount:    75,
		Model:     "test-model",
		Condition: `and(has-inviter(), github-star("zgsm-ai.zgsm"))`,
		Status:    true,
	}
	if err := ctx.StrategyService.CreateStrategy(strategy); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create strategy failed: %v", err)}
	}

	// Execute strategy
	users := []models.UserInfo{*user}
	ctx.StrategyService.ExecStrategy(strategy, users)

	// Check execution result
	var executeCount int64
	ctx.DB.Model(&models.QuotaExecute{}).Where("strategy_id = ? AND user_id = ? AND status = 'completed'", strategy.ID, user.ID).Count(&executeCount)

	if executeCount != 1 {
		return TestResult{Passed: false, Message: fmt.Sprintf("First execution expected 1 time, actually executed %d times", executeCount)}
	}

	// Verify strategy name in audit record (for invitation strategy, check inviter's audit record)
	if err := verifyStrategyNameInAudit(ctx, inviterUser.ID, "inviter-star-reward", models.OperationRecharge); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Strategy name verification failed: %v", err)}
	}

	// Second execute strategy (should be skipped because it has already been executed)
	ctx.StrategyService.ExecStrategy(strategy, users)

	// Check second execution result (should still be 1 time)
	ctx.DB.Model(&models.QuotaExecute{}).Where("strategy_id = ? AND user_id = ? AND status = 'completed'", strategy.ID, user.ID).Count(&executeCount)

	if executeCount != 1 {
		return TestResult{Passed: false, Message: fmt.Sprintf("Single strategy repeated execution, expected still 1 time, actually executed %d times", executeCount)}
	}

	return TestResult{Passed: true, Message: "Inviter Star Reward Strategy Test Succeeded"}
}

// testStrategyInviteUserStar test strategy invited user star reward (reward to invited user)
func testStrategyInviteUserStar(ctx *TestContext) TestResult {
	// Create test inviter user
	inviterUser := createTestInviterUser("invite_user_star_inviter_test", "Inviter User Star Test", 0, "")
	if err2 := ctx.DB.AuthDB.Create(inviterUser).Error; err2 != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create inviter user failed: %v", err2)}
	}

	// Create test invited user with GitHub star
	user := createTestInviterUser("invite_user_star_invited_test", "Invited User Star Test", 0, inviterUser.ID)
	if err := ctx.DB.AuthDB.Create(user).Error; err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create invited user failed: %v", err)}
	}

	// Create invited user star reward strategy (reward goes to invited user, not inviter)
	strategy := &models.QuotaStrategy{
		Name:      "invitee-star-reward",
		Title:     "Invitee Star Reward",
		Type:      "single",
		Amount:    25,
		Model:     "test-model",
		Condition: `and(has-inviter(), github-star("zgsm-ai.zgsm"))`,
		Status:    true,
	}
	if err := ctx.StrategyService.CreateStrategy(strategy); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Create strategy failed: %v", err)}
	}

	// Execute strategy
	users := []models.UserInfo{*user}
	ctx.StrategyService.ExecStrategy(strategy, users)

	// Check execution result
	var executeCount int64
	ctx.DB.Model(&models.QuotaExecute{}).Where("strategy_id = ? AND user_id = ? AND status = 'completed'", strategy.ID, user.ID).Count(&executeCount)

	if executeCount != 1 {
		return TestResult{Passed: false, Message: fmt.Sprintf("First execution expected 1 time, actually executed %d times", executeCount)}
	}

	// Verify strategy name in audit record (for invitee strategy, check invited user's audit record)
	if err := verifyStrategyNameInAudit(ctx, user.ID, "invitee-star-reward", models.OperationRecharge); err != nil {
		return TestResult{Passed: false, Message: fmt.Sprintf("Strategy name verification failed: %v", err)}
	}

	// Second execute strategy (should be skipped because it has already been executed)
	ctx.StrategyService.ExecStrategy(strategy, users)

	// Check second execution result (should still be 1 time)
	ctx.DB.Model(&models.QuotaExecute{}).Where("strategy_id = ? AND user_id = ? AND status = 'completed'", strategy.ID, user.ID).Count(&executeCount)

	if executeCount != 1 {
		return TestResult{Passed: false, Message: fmt.Sprintf("Single strategy repeated execution, expected still 1 time, actually executed %d times", executeCount)}
	}

	return TestResult{Passed: true, Message: "Invited User Star Reward Strategy Test Succeeded"}
}
