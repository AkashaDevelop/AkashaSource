package xuanjian

// ～宸汐玄鉴·镜中镜——MinHash 近似重复检测～ (・ω・)ノ
// 用三个独立哈希函数估算 Jaccard 相似度，
// 发现有人在疯狂试探"哪条 prompt 能绕过防护"的时候，
// 这面镜子就会亮起来 ✦

import (
	"strings"
	"unicode"
)

const (
	minHashK    = 3    // 哈希函数数量
	minHashSim  = 0.85 // 相似度阈值
	minHashScan = 2000 // 最多扫描前 N 个字符（性能控制）
)

// computeMinHash 对文本计算 K 个最小哈希值（4-gram shingle）
func computeMinHash(text string) [minHashK]uint64 {
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
		var result [minHashK]uint64
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
	var result [minHashK]uint64
	seeds := [minHashK]uint64{0x9e3779b9, 0x6c62272e, 0xd97f4a7c}
	for k, seed := range seeds {
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

// jaccardEstimate 估算两组 MinHash 的 Jaccard 相似度
func jaccardEstimate(a, b [minHashK]uint64) float64 {
	match := 0
	for i := range a {
		if a[i] == b[i] {
			match++
		}
	}
	return float64(match) / float64(minHashK)
}

// countSimilarPrompts 统计 hashes 中与 target 相似度超过阈值的数量
func countSimilarPrompts(hashes []uint64, target [minHashK]uint64, threshold float64) int {
	count := 0
	// hashes 存的是 pack(minHashK values) 的 XOR，用快速近似
	// 这里简化：直接存完整 [K]uint64，但 profile 里存的是 uint64 列表
	// 实际上 profile.PromptHashes 存的是 pack 值，需要 unpack 比较
	// → 改为直接存 [3]uint64 并用独立字段
	_ = hashes
	_ = target
	_ = threshold
	return count
}

// PackMinHash 将 [K]uint64 打包成单个 uint64（用于 profile 存储）
func PackMinHash(h [minHashK]uint64) uint64 {
	result := h[0]
	for i := 1; i < minHashK; i++ {
		result = result*0x517cc1b727220a95 ^ h[i]
	}
	return result
}

// CheckDuplicateInProfile 检测 profile 的 hash 缓冲中有多少与 target 相似
// 注意：打包后的 hash 只能做精确匹配，不能估算相似度
// 真正需要相似度检测时应存完整 MinHash，这里做近似：相同打包值视为高度相似
func CheckDuplicateInProfile(hashes []uint64, target uint64) int {
	count := 0
	for _, h := range hashes {
		if h == target {
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
