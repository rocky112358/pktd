// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package difficulty

import (
	"math/big"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/util"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/pcutil"
)

func bn256() *big.Int {
	out := big.NewInt(0)
	out.SetBit(out, 256, 1)
	return out
}

var bigOne = big.NewInt(1)

func workForTarget(target *big.Int) *big.Int {
	out := bn256()
	tarPlusOne := new(big.Int).Add(target, bigOne)
	out.Div(out, tarPlusOne)
	return out
}

func targetForWork(work *big.Int) *big.Int {
	out := bn256()
	if work.Sign() == 0 {
		// 0 work, min difficulty
		return out
	}
	out.Sub(out, work)
	out.Div(out, work)
	return out
}

func getEffectiveWorkRequirement(bnBlockHeaderWork, bnMinAnnWork *big.Int, annCount uint64) *big.Int {
	if bnMinAnnWork.Sign() == 0 || annCount == 0 {
		// there are no announcements or zero announcement work, set work to maximum
		return new(big.Int).Sub(bn256(), bigOne)
	}

	out := new(big.Int).Set(bnBlockHeaderWork)

	// out = out**3
	out.Mul(out, out)
	out.Mul(out, bnBlockHeaderWork)

	// out /= bnMinAnnWork
	out.Div(out, bnMinAnnWork)
	// out /= annCount
	out.Div(out, new(big.Int).SetUint64(annCount))

	return out
}

// GetEffectiveTarget gives the effective target to beat based on the target in the
// block header, the minimum work (highest target) of any announcement and the number
// of announcements which were mined with.
func GetEffectiveTarget(blockHeaderTarget uint32, minAnnTarget uint32, annCount uint64) uint32 {
	bnBlockHeaderTarget := CompactToBig(blockHeaderTarget)
	bnMinAnnTarget := CompactToBig(minAnnTarget)

	bnBlockHeaderWork := workForTarget(bnBlockHeaderTarget)
	bnMinAnnWork := workForTarget(bnMinAnnTarget)

	bnEffectiveWork := getEffectiveWorkRequirement(bnBlockHeaderWork, bnMinAnnWork, annCount)

	bnEffectiveTarget := targetForWork(bnEffectiveWork)
	effectiveTarget := BigToCompact(bnEffectiveTarget)

	if effectiveTarget > 0x207fffff {
		return 0x207fffff
	}
	return effectiveTarget
}

// IsOk will return true if the hash is ok given the target number
func IsOk(hash []byte, target uint32) bool {
	var r [32]byte
	copy(r[:], hash[:32])
	pcutil.Reverse(r[:])
	bh := new(big.Int).SetBytes(r[:])
	th := CompactToBig(target)
	return th.Cmp(bh) >= 0
}

// GetAgedAnnTarget returns the target which will be used for valuing the announcement.
// The minAnnWork committed in the coinbase must not be less work (higher number) than
// the highest (least work) aged target for any announcement mined in that block.
// If the announcement is not valid for adding to the block, return 0xffffffff
func GetAgedAnnTarget(target, annAgeBlocks uint32) uint32 {
	if annAgeBlocks < util.Conf_PacketCrypt_ANN_WAIT_PERIOD {
		// announcement is not ready yet
		return 0xffffffff
	}
	bnAnnTar := CompactToBig(target)
	if annAgeBlocks == util.Conf_PacketCrypt_ANN_WAIT_PERIOD {
		// fresh ann, no aging
		return BigToCompact(bnAnnTar)
	}
	annAgeBlocks -= util.Conf_PacketCrypt_ANN_WAIT_PERIOD
	bnAnnWork := workForTarget(bnAnnTar)
	bnAnnWork.Div(bnAnnWork, big.NewInt(int64(annAgeBlocks)))
	bnAnnAgedTar := targetForWork(bnAnnWork)
	out := BigToCompact(bnAnnAgedTar)
	if out > 0x207fffff {
		return 0xffffffff
	}
	return out
}

// IsAnnMinDiffOk is kind of a sanity check to make sure that the miner doesn't provide
// "silly" results which might trigger wrong behavior from the diff computation
func IsAnnMinDiffOk(target uint32) bool {
	if target == 0 || target > 0x20ffffff {
		return false
	}
	work := workForTarget(CompactToBig(target))
	return work.Sign() > 0 && work.Cmp(bn256()) < 0
}
