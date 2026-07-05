package xuanjian

// ～宸汐玄鉴·镜中镜——MinHash 近似重复检测～ (・ω・)ノ
// 用 K 个独立哈希函数估算 Jaccard 相似度，
// 发现有人在疯狂试探"哪条 prompt 能绕过防护"（哪怕每次都换几个字）的时候，
// 这面镜子就会亮起来 ✦
//
// 之前的实现有个坑：算完 K 个哈希后会 PackMinHash 压缩成一个 uint64 存进 profile，
// 压缩之后就只能判断"是否完全相等"，退化成了精确匹配——只要 prompt 有一个字不一样就漏检。
// 现在把完整的 K 维签名原样存进 profile，比较时才能真正估算 Jaccard 相似度。

import (
	"strings"
	"unicode"
)

const (
	minHashK    = 16   // 哈希函数数量，越多分辨率越细（相似度粒度是 1/minHashK）
	minHashSim  = 0.85 // 相似度阈值，判定"近似重复"
	minHashScan = 2000 // 最多扫描前 N 个字符（性能控制）
)

// MinHashSignature 一条 prompt 的完整 K 维 MinHash 签名
type MinHashSignature [minHashK]uint64

// IsZero 判断是不是空签名（没算过 / 没有内容）
func (s MinHashSignature) IsZero() bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}

// computeMinHash 对文本计算 K 个最小哈希值（4-gram shingle）
func computeMinHash(text string) MinHashSignature {
	runes := []rune(text)
	if len(runes) > minHashScan {
		runes = runes[:minHashScan]
	}
	text = strings.ToLower(string(runes))
	// 移除空白和标点，只保留字母/数字/CJK
	text = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, text)

	if len([]rune(text)) < 4 {
		// 文本太短，直接用全文哈希
		var result MinHashSignature
		h := fnv64([]byte(text))
		for i := range result {
			result[i] = h ^ (uint64(i) * 0x9e3779b97f4a7c15)
		}
		return result
	}

	// 生成 4-gram shingles
	runes = []rune(text)
	shingles := make([]uint64, 0, len(runes)-3)
	for i := 0; i <= len(runes)-4; i++ {
		s := string(runes[i : i+4])
		shingles = append(shingles, fnv64([]byte(s)))
	}

	// K 个最小哈希（每个种子对应一套哈希函数）
	var result MinHashSignature
	for k := 0; k < minHashK; k++ {
		seed := minHashSeed(k)
		minVal := ^uint64(0)
		for _, sh := range shingles {
			h := sh ^ seed
			h = (h ^ (h >> 30)) * 0xbf58476d1ce4e5b9
			h = (h ^ (h >> 27)) * 0x94d049bb133111eb
			h = h ^ (h >> 31)
			if h < minVal {
				minVal = h
			}
		}
		result[k] = minVal
	}
	return result
}

// minHashSeed 按下标派生第 k 个种子，不用手写 K 个常量～
func minHashSeed(k int) uint64 {
	seed := uint64(0x9e3779b97f4a7c15) + uint64(k)*0x517cc1b727220a95
	seed = (seed ^ (seed >> 30)) * 0xbf58476d1ce4e5b9
	return seed ^ (seed >> 27)
}

// jaccardEstimate 估算两组 MinHash 签名的 Jaccard 相似度
func jaccardEstimate(a, b MinHashSignature) float64 {
	match := 0
	for i := range a {
		if a[i] == b[i] {
			match++
		}
	}
	return float64(match) / float64(minHashK)
}

// countSimilarPrompts 统计 signatures 中与 target 相似度达到 threshold 的数量，
// 真正按 Jaccard 估算比较，字符级小差异（改一两个词）也能命中，不再要求完全相同～
func countSimilarPrompts(signatures []MinHashSignature, target MinHashSignature, threshold float64) int {
	count := 0
	for _, sig := range signatures {
		if jaccardEstimate(sig, target) >= threshold {
			count++
		}
	}
	return count
}

// fnv64 FNV-1a 64bit 哈希（轻量、无依赖）
func fnv64(data []byte) uint64 {
	var h uint64 = 14695981039346656037
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}
