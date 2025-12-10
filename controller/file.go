package controller

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// FileUploadRequest 文件上传请求结构
type FileUploadRequest struct {
	FileURL string `json:"file_url" binding:"omitempty,url"`
}

// UploadFile 文件上传控制器
func UploadFile(c *gin.Context) {
	// 尝试从表单获取文件
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		// 通过表单上传文件
		defer file.Close()

		// 获取文件扩展名
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			// 如果没有扩展名，尝试从Content-Type推断
			contentType := header.Header.Get("Content-Type")
			ext = getFileExtensionFromContentType(contentType)
			if ext == "" {
				ext = ".bin" // 默认使用bin扩展名
			}
		}

		// 生成唯一文件名
		filename := generateUniqueFilename() + ext
		fullPath := filepath.Join("./files", filename)

		// 创建目录（如果不存在）
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			log.Printf("创建目录失败: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "创建目录失败",
				"success": false,
			})
			return
		}

		// 保存文件
		outFile, err := os.Create(fullPath)
		if err != nil {
			log.Printf("创建文件失败: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "创建文件失败",
				"success": false,
			})
			return
		}
		defer outFile.Close()

		if _, err = io.Copy(outFile, file); err != nil {
			log.Printf("保存文件失败: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "保存文件失败",
				"success": false,
			})
			return
		}

		// 返回成功信息
		c.JSON(http.StatusOK, gin.H{
			"file_url": "/file/" + filename,
			"filename": filename,
			"success":  true,
		})
		return
	}

	// 尝试从JSON请求体获取file_url
	var req FileUploadRequest
	err = c.ShouldBindJSON(&req)
	if err == nil && req.FileURL != "" {
		// 通过URL下载文件
		// 创建一个自定义的HTTP客户端，跳过SSL证书验证，并处理重定向
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// 确保重定向请求也携带相同的请求头
				req.Header = via[0].Header
				return nil
			},
		}
		// 创建请求并添加适当的请求头
		httpReq, err := http.NewRequest("GET", req.FileURL, nil)
		if err != nil {
			log.Printf("创建请求失败: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "下载文件失败",
				"success": false,
			})
			return
		}
		// 添加浏览器类型的User-Agent
		httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		// 添加接受任何内容类型的Accept头
		httpReq.Header.Set("Accept", "*/*")
		// 添加Connection头
		httpReq.Header.Set("Connection", "keep-alive")
		// 发送请求
		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("下载文件失败: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "下载文件失败",
				"success": false,
			})
			return
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusOK, gin.H{
				"message": "文件下载失败，状态码: " + fmt.Sprintf("%d", resp.StatusCode),
				"success": false,
			})
			return
		}

		// 获取文件扩展名
		ext := filepath.Ext(req.FileURL)
		if ext == "" {
			// 尝试从Content-Disposition头获取文件名和扩展名
			disposition := resp.Header.Get("Content-Disposition")
			if disposition != "" {
				// 查找filename="..." 或 filename*="..."
				parts := strings.Split(disposition, ";")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if strings.HasPrefix(part, "filename=") {
						// 提取filename的值
						filenameValue := strings.TrimPrefix(part, "filename=")
						filenameValue = strings.Trim(filenameValue, `"'`)
						ext = filepath.Ext(filenameValue)
						break
					} else if strings.HasPrefix(part, "filename*") {
						// 提取filename*的值
						filenameValue := strings.TrimPrefix(part, "filename*")
						filenameValue = strings.TrimPrefix(filenameValue, "=")
						filenameValue = strings.Trim(filenameValue, `"'`)
						// 简单处理，直接提取扩展名
						ext = filepath.Ext(filenameValue)
						break
					}
				}
			}

			// 如果从Content-Disposition头没有获取到扩展名，尝试从Content-Type推断
			if ext == "" {
				contentType := resp.Header.Get("Content-Type")
				// 从image.go中复用getFileExtensionFromContentType函数的逻辑
				typeMap := map[string]string{
					"application/pdf":    ".pdf",
					"application/msword": ".doc",
					"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
					"application/vnd.ms-excel": ".xls",
					"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
					"application/vnd.ms-powerpoint":                                             ".ppt",
					"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
					"text/plain":                   ".txt",
					"application/zip":              ".zip",
					"application/x-rar-compressed": ".rar",
					"application/x-7z-compressed":  ".7z",
					"audio/mpeg":                   ".mp3",
					"audio/wav":                    ".wav",
					"audio/ogg":                    ".ogg",
					"video/mp4":                    ".mp4",
					"video/x-msvideo":              ".avi",
					"video/quicktime":              ".mov",
				}
				ext = typeMap[strings.ToLower(contentType)]
				if ext == "" {
					ext = ".bin" // 默认使用bin扩展名
				}
			}
		}

		// 生成唯一文件名
		filename := generateUniqueFilename() + ext
		fullPath := filepath.Join("./files", filename)

		// 创建目录（如果不存在）
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			log.Printf("创建目录失败: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "创建目录失败",
				"success": false,
			})
			return
		}

		// 保存文件
		outFile, err := os.Create(fullPath)
		if err != nil {
			log.Printf("创建文件失败: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "创建文件失败",
				"success": false,
			})
			return
		}
		defer outFile.Close()

		if _, err = io.Copy(outFile, resp.Body); err != nil {
			log.Printf("保存文件失败: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "保存文件失败",
				"success": false,
			})
			return
		}

		// 返回成功信息
		c.JSON(http.StatusOK, gin.H{
			"file_url": "/file/" + filename,
			"filename": filename,
			"success":  true,
		})
		return
	}

	// 两种方式都失败
	c.JSON(http.StatusOK, gin.H{
		"message": "获取文件失败，请检查请求格式",
		"success": false,
	})
}

// GetFile 获取文件控制器
func GetFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusOK, gin.H{
			"message": "文件名不能为空",
			"success": false,
		})
		return
	}

	// 安全检查，防止路径遍历
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		c.JSON(http.StatusOK, gin.H{
			"message": "无效的文件名",
			"success": false,
		})
		return
	}

	fullPath := filepath.Join("./files", filename)

	// 检查文件是否存在
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{
				"message": "文件不存在",
				"success": false,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "访问文件失败",
				"success": false,
			})
		}
		return
	}

	// 根据文件扩展名设置正确的Content-Type
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := getContentTypeFromExtension(ext)
	c.Header("Content-Type", contentType)

	// 提供文件下载
	c.File(fullPath)
}

// 辅助函数：根据文件扩展名获取Content-Type
func getContentTypeFromExtension(ext string) string {
	extMap := map[string]string{
		// 图片类型
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".tiff": "image/tiff",
		// 文档类型
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".md":   "text/markdown",
		// 压缩文件类型
		".zip": "application/zip",
		".rar": "application/x-rar-compressed",
		".7z":  "application/x-7z-compressed",
		// 代码文件类型
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".go":   "text/x-go",
		".py":   "text/x-python",
		".java": "text/x-java",
		".c":    "text/x-csrc",
		".cpp":  "text/x-c++src",
		".cs":   "text/x-csharp",
		// 音频类型
		".mp3": "audio/mpeg",
		".wav": "audio/wav",
		".ogg": "audio/ogg",
		// 视频类型
		".mp4":  "video/mp4",
		".avi":  "video/avi",
		".mpeg": "video/mpeg",
		".webm": "video/webm",
		".mov":  "video/quicktime",
	}
	contentType, exists := extMap[ext]
	if !exists {
		return "application/octet-stream" // 默认返回二进制流
	}
	return contentType
}

// CleanupExpiredFiles 定时清理过期文件
func CleanupExpiredFiles() {
	// 每天执行一次清理
	cleanupInterval := 24 * time.Hour
	// 文件有效期为1天
	fileLifetime := 24 * time.Hour

	for {
		log.Println("开始清理过期文件...")
		filesDir := "./files"

		// 遍历files目录
		err := filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("遍历文件失败: %v", err)
				return nil
			}

			// 跳过目录
			if info.IsDir() {
				return nil
			}

			// 检查文件是否过期
			if time.Since(info.ModTime()) > fileLifetime {
				// 删除过期文件
				if err := os.Remove(path); err != nil {
					log.Printf("删除过期文件失败 %s: %v", path, err)
				} else {
					log.Printf("已删除过期文件: %s", path)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("清理过期文件时发生错误: %v", err)
		}

		log.Println("过期文件清理完成")

		// 等待下一次清理
		time.Sleep(cleanupInterval)
	}
}
