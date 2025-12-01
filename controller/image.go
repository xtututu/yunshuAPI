package controller

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
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

// ImageUploadRequest 图片上传请求结构
type ImageUploadRequest struct {
	ImageURL string `json:"image_url" binding:"required,url"`
}

// UploadImage 上传图片控制器
func UploadImage(c *gin.Context) {
	var req ImageUploadRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "无效的参数格式",
			"success": false,
		})
		return
	}

	// 验证URL是否为图片
	if !isValidImageURL(req.ImageURL) {
		c.JSON(http.StatusOK, gin.H{
			"message": "请提供有效的图片URL",
			"success": false,
		})
		return
	}

	// 下载图片
	resp, err := http.Get(req.ImageURL)
	if err != nil {
		log.Printf("下载图片失败: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "下载图片失败",
			"success": false,
		})
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, gin.H{
			"message": "图片下载失败，状态码: " + fmt.Sprintf("%d", resp.StatusCode),
			"success": false,
		})
		return
	}

	// 获取文件扩展名
	ext := filepath.Ext(req.ImageURL)
	if ext == "" {
		// 如果没有扩展名，尝试从Content-Type推断
		contentType := resp.Header.Get("Content-Type")
		ext = getFileExtensionFromContentType(contentType)
		if ext == "" {
			ext = ".jpg" // 默认使用jpg
		}
	}

	// 生成唯一文件名
	filename := generateUniqueFilename() + ext
	fullPath := filepath.Join("./images", filename)

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
		"message":   "图片上传成功",
		"success":   true,
		"image_url": "/image/" + filename,
		"filename":  filename,
	})
}

// GetImage 获取图片控制器
func GetImage(c *gin.Context) {
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

	fullPath := filepath.Join("./images", filename)

	// 检查文件是否存在
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{
				"message": "图片不存在",
				"success": false,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "访问图片失败",
				"success": false,
			})
		}
		return
	}

	// 读取文件内容
	fileData, err := os.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "读取图片失败",
			"success": false,
		})
		return
	}

	// 设置Content-Length头
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileData)))

	// 根据文件扩展名设置正确的Content-Type
	ext := strings.ToLower(filepath.Ext(filename))
	var contentType string
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".bmp":
		contentType = "image/bmp"
	default:
		contentType = "image/jpeg" // 默认设置为jpeg
	}

	// 使用c.Data直接发送数据和Content-Type，确保Content-Type正确设置
	c.Data(http.StatusOK, contentType, fileData)
}

// 辅助函数：验证URL是否为图片
func isValidImageURL(url string) bool {
	lowerURL := strings.ToLower(url)
	validExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
	for _, ext := range validExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}
	return false
}

// 辅助函数：根据Content-Type获取文件扩展名
func getFileExtensionFromContentType(contentType string) string {
	typeMap := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
		"image/bmp":  ".bmp",
	}
	return typeMap[strings.ToLower(contentType)]
}

// 辅助函数：生成唯一文件名
func generateUniqueFilename() string {
	// 使用时间戳和随机字符串生成唯一文件名
	timestamp := time.Now().UnixNano()
	hasher := md5.New()
	hasher.Write([]byte(fmt.Sprintf("%d", timestamp)))
	return hex.EncodeToString(hasher.Sum(nil))
}
