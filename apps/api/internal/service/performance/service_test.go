func TestService_GetUserAnalytics_ReturnsEmptyForNoVaults(t *testing.T) {
	repo := newFakeRepo()
	vaultRepo := &stubVaultRepository{vaults: []vault.Vault{}, err: nil}
	svc := NewService(repo, vaultRepo)

	ctx := context.Background()
	userID := uuid.New()
	resp, err := svc.GetUserAnalytics(ctx, userID, time.Now().AddDate(0, 0, -30), time.Now())
	if err != nil {
		t.Fatalf("GetUserAnalytics: %v", err)
	}

	if len(resp.DailySnapshots) != 0 {
		t.Fatalf("expected empty daily snapshots, got %d", len(resp.DailySnapshots))
	}
	if len(resp.VaultMonthlyYield) != 0 {
		t.Fatalf("expected empty vault monthly yield, got %d", len(resp.VaultMonthlyYield))
	}
	if len(resp.CurrentAllocation) != 0 {
		t.Fatalf("expected empty current allocation, got %d", len(resp.CurrentAllocation))
	}
	if len(resp.Vaults) != 0 {
		t.Fatalf("expected empty vaults, got %d", len(resp.Vaults))
	}
}

func TestService_GetUserAnalytics_WithVaults(t *testing.T) {
	now := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	id1 := uuid.New()
	id2 := uuid.New()

	// Mock vault repository to return two vaults
	vaultRepo := &stubVaultRepository{
		vaults: []vault.Vault{
			{
				ID:          id1,
				UserID:      uuid.New(),
				ContractAddress: "VAULT1",
				TotalDeposited: decimal.NewFromInt(1000),
				CurrentBalance: decimal.NewFromInt(1100),
				Currency:     "USD",
				YieldEarned:  decimal.NewFromInt(100),
				Allocations: []vault.Allocation{
					{
						Protocol: "Aave",
						Amount:   decimal.NewFromInt(600),
						APY:      decimal.NewFromFloat(0.08), // 8%
					},
					{
						Protocol: "Blend",
						Amount:   decimal.NewFromInt(400),
						APY:      decimal.NewFromFloat(0.12), // 12%
					},
				},
			},
			{
				ID:          id2,
				UserID:      uuid.New(),
				ContractAddress: "VAULT2",
				TotalDeposited: decimal.NewFromInt(500),
				CurrentBalance: decimal.NewFromInt(550),
				Currency:     "USD",
				YieldEarned:  decimal.NewFromInt(50),
				Allocations: []vault.Allocation{
					{
						Protocol: "Compound",
						Amount:   decimal.NewFromInt(500),
						APY:      decimal.NewFromFloat(0.06), // 6%
					},
				},
			},
		},
		err: nil,
	}

	// Mock performance repository to return some snapshots for history
	repo := newFakeRepo()
	// Add a snapshot for the first vault to have some history
	snapshot := perfdom.Snapshot{
		ID:             uuid.New(),
		VaultID:        id1,
		TotalBalance:   decimal.NewFromInt(1050),
		TotalDeposited: decimal.NewFromInt(1000),
		TotalYieldEarned: decimal.NewFromInt(50),
		SharePrice:     decimal.NewFromInt(1),
		SnapshotAt:     now.AddDate(0, 0, -15), // 15 days ago
		AllocationBreakdown: []perfdom.AllocationBreakdownEntry{
			{Source: "Aave", Amount: decimal.NewFromInt(600), APY: decimal.NewFromFloat(0.08)},
			{Source: "Blend", Amount: decimal.NewFromInt(400), APY: decimal.NewFromFloat(0.12)},
		},
	}
	repo.snapshots = append(repo.snapshots, snapshot)

	svc := NewService(repo, vaultRepo)

	ctx := context.Background()
	userID := vaultRepo.vaults[0].UserID // Assuming both vaults belong to same user for test
	fromTime := now.AddDate(0, 0, -30)
	toTime := now

	resp, err := svc.GetUserAnalytics(ctx, userID, fromTime, toTime)
	if err != nil {
		t.Fatalf("GetUserAnalytics: %v", err)
	}

	// Check that we got some daily snapshots
	if len(resp.DailySnapshots) == 0 {
		t.Fatalf("expected at least one daily snapshot")
	}

	// Check current allocation - should have 3 protocols (Aave, Blend, Compound)
	if len(resp.CurrentAllocation) != 3 {
		t.Fatalf("expected 3 protocol allocations, got %d", len(resp.CurrentAllocation))
	}

	// Check vaults info - should have 2 vaults
	if len(resp.Vaults) != 2 {
		t.Fatalf("expected 2 vaults, got %d", len(resp.Vaults))
	}

	// Check performance metrics
	if resp.PerformanceMetrics.TotalYieldEarned != 150 { // 100 from vault1 + 50 from vault2
		t.Fatalf("expected total yield earned 150, got %.2f", resp.PerformanceMetrics.TotalYieldEarned)
	}
	if resp.PerformanceMetrics.TotalDeposited != 1500 { // 1000 + 500
		t.Fatalf("expected total deposited 1500, got %.2f", resp.PerformanceMetrics.TotalDeposited)
	}
	if resp.PerformanceMetrics.NetPosition != 150 { // (1100+550) - (1000+500) = 150
		t.Fatalf("expected net position 150, got %.2f", resp.PerformanceMetrics.NetPosition)
	}
}