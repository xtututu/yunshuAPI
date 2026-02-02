# 部署指南：在2GB内存服务器上运行yunshuAPI

## 问题分析

在2GB内存的服务器上直接运行 `docker-compose build && docker-compose up -d` 会导致内存不足卡死，主要原因：

1. **构建过程内存消耗大**：Docker构建过程中Go编译需要大量内存
2. **运行时内存使用高**：应用默认配置可能超出服务器内存限制
3. **前端静态文件占用**：嵌入在二进制中的前端文件增加了内存使用

## 优化方案

### 1. 内存限制配置

已在 `docker-compose.yml` 中添加了内存限制：

```yaml
# 内存限制配置
mem_limit: 768m
mem_reservation: 512m
```

### 2. 本地构建方案

使用 `build_and_deploy.sh` 脚本在本地构建镜像，然后上传到服务器运行，避免在内存有限的服务器上进行构建操作。

### 3. Dockerfile 优化

- 添加了内存使用限制：`ENV GOMEMLIMIT=512MiB`
- 使用更紧凑的编译选项：`-trimpath -ldflags "-s -w"`
- 使用轻量的 Alpine 基础镜像

### 4. Go 应用优化

- 生产模式运行：`gin.SetMode(gin.ReleaseMode)`
- 减少不必要的后台任务
- 优化内存缓存使用

## 部署步骤

### 步骤1：配置本地构建脚本

1. 编辑 `build_and_deploy.sh` 文件，设置服务器信息：

```bash
# 服务器配置
SERVER_IP="your-server-ip"
SERVER_USER="root"
```

2. 为脚本添加执行权限：

```bash
chmod +x build_and_deploy.sh
```

### 步骤2：本地构建并部署

在本地运行构建脚本：

```bash
./build_and_deploy.sh
```

脚本会自动：
1. 在本地构建镜像
2. 保存为tar文件
3. 上传到服务器
4. 在服务器上加载镜像并运行
5. 清理临时文件

### 步骤3：验证部署

部署完成后，检查容器状态：

```bash
# 在服务器上执行
docker ps -a | grep new-api
```

检查应用是否正常运行：

```bash
curl http://localhost:3000/api/status
```

### 步骤4：监控内存使用

使用以下命令监控服务器内存使用情况：

```bash
# 实时监控
top

# 或查看Docker容器内存使用
docker stats new-api
```

## 内存调整建议

如果部署后仍然出现内存问题，可以根据实际情况调整以下参数：

### 1. 调整容器内存限制

编辑 `docker-compose.yml` 中的内存限制：

```yaml
# 内存限制配置
mem_limit: 512m  # 进一步减少上限
mem_reservation: 384m  # 进一步减少预留
```

### 2. 禁用不必要的功能

在服务器上创建 `.env` 文件，禁用一些内存密集型功能：

```env
# 禁用内存缓存
MEMORY_CACHE_ENABLED=false

# 减少同步频率
SYNC_FREQUENCY=300

# 禁用批量更新
BATCH_UPDATE_ENABLED=false
```

### 3. 优化Go应用参数

可以通过环境变量调整Go应用的内存使用：

```env
# 限制Go垃圾回收器
GOGC=80

# 限制HTTP客户端连接数
HTTP_CLIENT_MAX_IDLE_CONNECTIONS=10
HTTP_CLIENT_MAX_IDLE_CONNECTIONS_PER_HOST=5
```

## 故障排查

### 1. 构建失败

如果本地构建失败，检查：
- 本地Docker环境是否正常
- 网络连接是否稳定
- Go依赖是否可以正常下载

### 2. 部署失败

如果部署失败，检查：
- 服务器网络连接是否正常
- 服务器Docker环境是否正常
- 服务器磁盘空间是否足够

### 3. 运行时内存不足

如果运行时出现内存不足，检查：
- 容器内存限制是否设置合理
- 是否有内存泄漏问题
- 是否有过多的并发请求

### 4. 应用响应缓慢

如果应用响应缓慢，检查：
- 服务器CPU使用情况
- 数据库查询性能
- 网络连接质量

## 最佳实践

1. **定期清理日志**：避免日志文件占用过多磁盘空间
2. **监控系统状态**：使用监控工具定期检查服务器状态
3. **合理配置并发**：根据服务器配置调整应用并发参数
4. **定期更新应用**：保持应用和依赖的最新版本
5. **备份重要数据**：定期备份数据库和配置文件

## 总结

通过以上优化措施，yunshuAPI 应该能够在2GB内存的服务器上正常运行。关键是使用本地构建方案避免服务器构建压力，并合理设置内存限制以适应服务器配置。

如果遇到任何问题，请参考故障排查部分或联系技术支持。
