package ssz_static

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/ssz_static"
)

func TestMinimal_Electra_SSZStatic(t *testing.T) {
	ssz_static.RunSSZStaticTests(t, "minimal")
}
