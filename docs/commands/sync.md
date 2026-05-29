# sync - 同步密钥

将敏感文件同步到密钥管理服务。

## 语法

```bash
yews sync [选项]
yews sync pull [选项]
```

## 子命令

### sync (push)

推送密钥到密钥管理服务。

```bash
yews sync \
  --provider infisical \
  --project-id <project-id> \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt
```

### sync pull

从密钥管理服务拉取密钥到本地文件。

```bash
yews sync pull \
  --provider infisical \
  --project-id <project-id> \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt
```

## 选项

### --provider, -p

密钥管理服务提供商。

默认值：`infisical`

目前支持：
- `infisical` - Infisical 密钥管理服务

```bash
yews sync --provider infisical
```

### --project-id

Infisical 项目 ID。

```bash
yews sync --project-id abc123def456
```

### --env, --environment

提供商中的环境名称。

```bash
yews sync --env dev
yews sync --env prod
```

### --path

提供商中的路径/文件夹（例如 /yewseal）。

```bash
yews sync --path /yewseal
```

### --name, -n

提供商中的密钥名称。

默认值：`AGE_KEY_FILE`

```bash
yews sync --name AGE_KEY_FILE
```

### --key-file, -k

要同步的密钥文件路径。

默认值：`.age/keys.txt`

```bash
yews sync --key-file .age/keys.txt
```

## Infisical 配置

### 前置要求

1. 安装 Infisical CLI：
   ```bash
   # macOS
   brew install infisical/get-cli/infisical
   
   # Linux
   curl -1sLf 'https://dl.cloudsmith.io/public/infisical/infisical-cli/setup.deb.sh' | sudo -E bash
   sudo apt-get update && sudo apt-get install -y infisical
   ```

2. 登录 Infisical：
   ```bash
   infisical login
   ```

### 获取项目 ID

在 Infisical 控制台中：
1. 进入项目设置
2. 复制项目 ID

或使用 CLI：
```bash
infisical projects list
```

## 示例

### 推送密钥到 Infisical

```bash
# 推送到开发环境
yews sync \
  --provider infisical \
  --project-id abc123def456 \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt

# 推送到生产环境
yews sync \
  --provider infisical \
  --project-id abc123def456 \
  --env prod \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt
```

### 从 Infisical 拉取密钥

```bash
# 从开发环境拉取
yews sync pull \
  --provider infisical \
  --project-id abc123def456 \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt

# 拉取到不同位置
yews sync pull \
  --provider infisical \
  --project-id abc123def456 \
  --env prod \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file ~/.age/prod-key.txt
```

## 团队协作工作流

### 初始化项目（项目负责人）

```bash
# 1. 初始化项目
yews init

# 2. 推送密钥到 Infisical
yews sync \
  --provider infisical \
  --project-id <project-id> \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE

# 3. 提交加密文件到 Git
git add .sops.yaml *.enc.toml.yaml
git commit -m "Add encrypted configs"
git push
```

### 加入项目（团队成员）

```bash
# 1. 克隆仓库
git clone <repo-url>
cd <repo>

# 2. 从 Infisical 拉取密钥
yews sync pull \
  --provider infisical \
  --project-id <project-id> \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE

# 3. 解密配置文件
yews decrypt -i config.enc.toml.yaml -o config.toml
```

## CI/CD 集成

### GitHub Actions

```yaml
name: Deploy
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Infisical CLI
        run: |
          curl -1sLf 'https://dl.cloudsmith.io/public/infisical/infisical-cli/setup.deb.sh' | sudo -E bash
          sudo apt-get update && sudo apt-get install -y infisical
      
      - name: Pull secrets
        env:
          INFISICAL_TOKEN: ${{ secrets.INFISICAL_TOKEN }}
        run: |
          yews sync pull \
            --provider infisical \
            --project-id ${{ secrets.INFISICAL_PROJECT_ID }} \
            --env prod \
            --path /yewseal \
            --name AGE_KEY_FILE
      
      - name: Decrypt configs
        run: yews decrypt -i config.enc.toml.yaml -o config.toml
```

## 注意事项

1. **认证要求**：使用前需要先通过 `infisical login` 登录
2. **权限管理**：确保团队成员有对应环境的访问权限
3. **密钥安全**：拉取的密钥文件应添加到 `.gitignore`
4. **环境隔离**：建议为不同环境使用不同的密钥

## 相关命令

- [init](/commands/init) - 初始化项目并生成密钥
- [encrypt](/commands/encrypt) - 加密配置文件
- [decrypt](/commands/decrypt) - 解密配置文件
