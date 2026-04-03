package utils

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	reUrl           = regexp.MustCompile(`(https?|ftp|file)://[-A-Za-z0-9+&@#/%?=~_|!:,.;]+[-A-Za-z0-9+&@#/%=~_|]`)
	reWinNonSupport = regexp.MustCompile(`[/\\:*?"<>\|]`)
)

func PathExists(path string) (bool, error) {
	_, err := os.Lstat(path)

	if err == nil {

		return true, nil

	}

	if os.IsNotExist(err) {

		return false, nil

	}
	return false, err
}

// reserve 5 bytes for suffix
// NTFS 单个文件名硬限制为 255 字符（与长路径设置无关）
const DefaultMaxFileNameLen = 155

// 可配置的文件名长度限制（可在 main.go 中修改）
var MaxFileNameLen = DefaultMaxFileNameLen

// 将无后缀的文件名更新为有效的 Windows 文件名
func WinFileName(name string) string {
	// 将字节切片转换为字符串
	// 使用正则表达式进行替换
	name = reUrl.ReplaceAllString(name, "")
	name = reWinNonSupport.ReplaceAllString(name, "")

	// 创建一个缓冲区，避免多次分配
	var buffer bytes.Buffer

	// 遍历字符串，对字符进行处理
	for _, ch := range name {
		switch ch {
		case '\r':
			// 跳过 \r
			continue
		case '\n':
			// 将 \n 替换为空格
			if buffer.Len()+1 > MaxFileNameLen {
				break
			}
			buffer.WriteRune(' ')
		default:
			// 其他字符直接添加到缓冲区
			if buffer.Len()+utf8.RuneLen(ch) > MaxFileNameLen {
				break
			}
			buffer.WriteRune(ch)
		}
	}

	return buffer.String()
}

func UniquePath(path string) (string, error) {
	for {
		exist, err := PathExists(path)
		if err != nil {
			return "", err
		}
		if !exist {
			return path, nil
		}

		dir := filepath.Dir(path)
		base := filepath.Base(path)
		ext := filepath.Ext(path)
		stem, _ := strings.CutSuffix(base, ext)
		stemlen := len(stem)

		// 处理已括号结尾的文件名
		if stemlen > 0 && stem[stemlen-1] == ')' {
			if left := strings.LastIndex(stem, "("); left != -1 {

				index, err := strconv.Atoi(stem[left+1 : stemlen-1])
				if err == nil {
					index++
					stem = fmt.Sprintf("%s(%d)", stem[:left], index)
					path = filepath.Join(dir, stem+ext)
					continue
				}
			}
		}

		path = filepath.Join(dir, stem+"(1)"+ext)
	}
}

// TweetFileName 生成统一的推文相关文件名
// 格式: { sanitized_text }_{ tweet_id }(序号).{ ext }
// 参数:
//   - text: 推文文本
//   - tweetId: 推文ID
//   - ext: 文件扩展名(包含点, 如 ".json", ".jpg")
// 返回:
//   - 基础文件名(不含序号)
// 命名规则:
//   1. 文本经过 WinFileName 清理
//   2. 完整文本(受MaxFileNameLen限制) + _ + tweet_id
//   3. 如果超长,优先截断文本,保留完整ID
//   4. 使用 UniquePath 添加序号(1), (2)等
// 示例:
//   TweetFileName("比基尼", 1355100264760393735, ".jpg") -> "比基尼_1355100264760393735.jpg"
func TweetFileName(text string, tweetId uint64, ext string) string {
	// 清理文本
	sanitizedText := WinFileName(text)

	// 构建ID部分
	idPart := fmt.Sprintf("_%d", tweetId)

	// 计算可用文本长度(预留5字节给序号(1))
	maxTextLen := MaxFileNameLen - len(idPart) - len(ext) - 5
	if maxTextLen < 0 {
		maxTextLen = 0
	}

	// 截断文本(如果超长)
	if len(sanitizedText) > maxTextLen {
		sanitizedText = sanitizedText[:maxTextLen]
	}

	// 如果文本为空,使用默认值
	if sanitizedText == "" {
		sanitizedText = "tweet"
	}

	// 组合文件名
	return sanitizedText + idPart + ext
}

func GetExtFromUrl(u string) (string, error) {
	pu, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return filepath.Ext(pu.Path), nil
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
