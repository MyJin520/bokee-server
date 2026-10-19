package randx

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand"
	"strings"
	"time"
)

// RandomInt 返回一个非负伪随机整数。
func RandomInt() int {
	return mrand.Int()
}

// RandomIntRange 返回指定范围内的伪随机整数 [min, max]。
func RandomIntRange(min, max int) int {
	if min > max {
		min, max = max, min
	}
	return mrand.Intn(max-min+1) + min
}

// RandomFloat64 返回 [0.0, 1.0) 范围内的伪随机浮点数。
func RandomFloat64() float64 {
	return mrand.Float64()
}

// RandomFloat64Range 返回指定范围内的伪随机浮点数 [min, max)。
func RandomFloat64Range(min, max float64) float64 {
	if min > max {
		min, max = max, min
	}
	return min + mrand.Float64()*(max-min)
}

// RandomBool 返回伪随机的布尔值。
func RandomBool() bool {
	return mrand.Intn(2) == 0
}

// RandomString 返回指定长度的伪随机字母数字字符串。
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[mrand.Intn(len(charset))]
	}
	return string(result)
}

// RandomDigitCode 返回指定长度的纯数字伪随机字符串。
func RandomDigitCode(length int) string {
	const digits = "0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = digits[mrand.Intn(len(digits))]
	}
	return string(result)
}

// RandomChoice 从切片中伪随机选出一个元素。
func RandomChoice[T any](slice []T) T {
	if len(slice) == 0 {
		var zero T
		return zero
	}
	return slice[mrand.Intn(len(slice))]
}

// RandomSample 从切片中伪随机选出 n 个不重复的元素。
func RandomSample[T any](slice []T, n int) []T {
	if n > len(slice) {
		n = len(slice)
	}
	indices := mrand.Perm(len(slice))
	result := make([]T, n)
	for i := 0; i < n; i++ {
		result[i] = slice[indices[i]]
	}
	return result
}

// Shuffle 返回切片的一个伪随机洗牌副本（Fisher-Yates 算法）。
func Shuffle[T any](slice []T) []T {
	result := make([]T, len(slice))
	copy(result, slice)
	for i := len(result) - 1; i > 0; i-- {
		j := mrand.Intn(i + 1)
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// WeightedRandomChoice 根据给定的权重执行一次伪随机加权选择。
func WeightedRandomChoice(items []string, weights []float64) string {
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}
	r := mrand.Float64() * totalWeight
	cumulative := 0.0
	for i, w := range weights {
		cumulative += w
		if r <= cumulative {
			return items[i]
		}
	}
	return items[len(items)-1]
}

// WeightedPicker 高效的加权随机选择器，内部使用累积权重和二分查找。
type WeightedPicker[T any] struct {
	items       []T
	cumWeights  []float64
	totalWeight float64
}

// NewWeightedPicker 根据给定的项目和权重创建一个 WeightedPicker。
func NewWeightedPicker[T any](items []T, weights []float64) *WeightedPicker[T] {
	wp := &WeightedPicker[T]{
		items:      items,
		cumWeights: make([]float64, len(items)),
	}
	cumulative := 0.0
	for i, w := range weights {
		cumulative += w
		wp.cumWeights[i] = cumulative
	}
	wp.totalWeight = cumulative
	return wp
}

// Pick 从 WeightedPicker 中执行一次伪随机加权选择。
func (wp *WeightedPicker[T]) Pick() T {
	r := mrand.Float64() * wp.totalWeight
	low, high := 0, len(wp.cumWeights)-1
	for low < high {
		mid := (low + high) / 2
		if wp.cumWeights[mid] < r {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return wp.items[low]
}

// SecureRandomInt 安全随机整数（crypto/rand）
func SecureRandomInt(min, max int64) (int64, error) {
	if min > max {
		min, max = max, min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return 0, err
	}
	return n.Int64() + min, nil
}

// SecureRandomBytes 生成指定长度的安全随机字节序列。
func SecureRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	return bytes, err
}

// SecureRandomToken 返回指定字节长度的安全随机十六进制 Token。
func SecureRandomToken(length int) (string, error) {
	bytes, err := SecureRandomBytes(length)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

// RandomGaussian 返回一个符合高斯（正态）分布的随机数，使用 Box-Muller 变换。
func RandomGaussian(mean, stddev float64) float64 {
	u1 := mrand.Float64()
	u2 := mrand.Float64()
	for u1 == 0 {
		u1 = mrand.Float64()
	}
	z0 := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
	return mean + z0*stddev
}

// RandomExponential 返回一个符合指数分布的随机数。
func RandomExponential(lambda float64) float64 {
	u := mrand.Float64()
	for u == 0 {
		u = mrand.Float64()
	}
	return -math.Log(u) / lambda
}

// RepeatableRandom 可重复的随机数生成器，使用固定种子。
type RepeatableRandom struct {
	rng *mrand.Rand
}

// NewRepeatableRandom 根据种子创建一个可重复的随机数生成器。
func NewRepeatableRandom(seed int64) *RepeatableRandom {
	return &RepeatableRandom{
		rng: mrand.New(mrand.NewSource(seed)),
	}
}

// Intn 返回 [0, n) 范围内的伪随机整数。
func (rr *RepeatableRandom) Intn(n int) int {
	return rr.rng.Intn(n)
}

// Float64 返回 [0.0, 1.0) 范围内的伪随机浮点数。
func (rr *RepeatableRandom) Float64() float64 {
	return rr.rng.Float64()
}

// SafeRandom 并发安全的伪随机数生成器。
type SafeRandom struct {
	rng *mrand.Rand
}

// NewSafeRandom 创建一个使用当前时间纳秒作为种子的并发安全随机数生成器。
func NewSafeRandom() *SafeRandom {
	return &SafeRandom{
		rng: mrand.New(mrand.NewSource(time.Now().UnixNano())),
	}
}

// Intn 返回 [0, n) 范围内的并发安全伪随机整数。
func (sr *SafeRandom) Intn(n int) int {
	return sr.rng.Intn(n)
}

// Float64 返回 [0.0, 1.0) 范围内的并发安全伪随机浮点数。
func (sr *SafeRandom) Float64() float64 {
	return sr.rng.Float64()
}

// RandomUUID 生成一个符合 v4 规范的安全随机 UUID 字符串。
func RandomUUID() string {
	bytes, _ := SecureRandomBytes(16)
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(bytes[0:4]),
		binary.BigEndian.Uint16(bytes[4:6]),
		binary.BigEndian.Uint16(bytes[6:8]),
		binary.BigEndian.Uint16(bytes[8:10]),
		bytes[10:16],
	)
}

// GeneratePassword 根据指定的字符集选项生成伪随机密码。
func GeneratePassword(length int, useUpper, useLower, useDigits, useSymbols bool) string {
	var charset strings.Builder
	if useUpper {
		charset.WriteString("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	}
	if useLower {
		charset.WriteString("abcdefghijklmnopqrstuvwxyz")
	}
	if useDigits {
		charset.WriteString("0123456789")
	}
	if useSymbols {
		charset.WriteString("!@#$%^&*()_+-=[]{}|;:,.<>?")
	}
	if charset.Len() == 0 {
		charset.WriteString("abcdefghijklmnopqrstuvwxyz")
	}
	chars := charset.String()
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[mrand.Intn(len(chars))]
	}
	return string(result)
}
