package utils

// UserIDExtractor 用于从用户结构体提取 ID 的泛型函数
type UserIDExtractor[T any] func(T) uint64

// ExtractIDs 通用 ID 提取函数
// 传入用户列表和提取函数，返回 ID 列表
func ExtractIDs[T any](users []T, extract UserIDExtractor[T]) []uint64 {
	if len(users) == 0 {
		return nil
	}
	uids := make([]uint64, len(users))
	for i, u := range users {
		uids[i] = extract(u)
	}
	return uids
}
