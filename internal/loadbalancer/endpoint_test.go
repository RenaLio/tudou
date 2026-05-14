package loadbalancer

import "testing"

func TestEndpointUpdateMetricsSkipsTTFTForNonStreamSuccess(t *testing.T) {
	ep := &Endpoint{
		EmaTTFT:        1600,
		EmaTPS:         100,
		EmaSuccessRate: 1.0,
	}

	ep.UpdateMetrics(true, false, 800, 120)

	if ep.EmaTTFT != 1600 {
		t.Fatalf("unexpected ema ttft, got=%v want=1600", ep.EmaTTFT)
	}
	if ep.EmaTPS != 104 {
		t.Fatalf("unexpected ema tps, got=%v want=104", ep.EmaTPS)
	}
	if ep.EmaSuccessRate != 1.0 {
		t.Fatalf("unexpected ema success rate, got=%v want=1.0", ep.EmaSuccessRate)
	}
}

func TestEndpointUpdateMetricsUpdatesTTFTForStreamSuccess(t *testing.T) {
	ep := &Endpoint{
		EmaTTFT:        1600,
		EmaTPS:         100,
		EmaSuccessRate: 1.0,
	}

	ep.UpdateMetrics(true, true, 800, 120)

	if ep.EmaTTFT != 1440 {
		t.Fatalf("unexpected ema ttft, got=%v want=1440", ep.EmaTTFT)
	}
	if ep.EmaTPS != 104 {
		t.Fatalf("unexpected ema tps, got=%v want=104", ep.EmaTPS)
	}
	if ep.EmaSuccessRate != 1.0 {
		t.Fatalf("unexpected ema success rate, got=%v want=1.0", ep.EmaSuccessRate)
	}
}
