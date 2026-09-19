# Cuetiy PostgreSQL 部署与数据迁移

> 代码侧已支持 `DB_DRIVER=postgres | mysql | sqlite`。  
> 云端默认 **PostgreSQL**；本文只覆盖：**服务器装 PG → 写配置 → 启动 →（可选）从 MySQL 搬数据**。

---

## 1. 服务器安装 PostgreSQL

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y postgresql postgresql-contrib
sudo systemctl enable --now postgresql
```

### CentOS / RHEL / Rocky

```bash
sudo dnf install -y postgresql-server postgresql-contrib
sudo postgresql-setup --initdb
sudo systemctl enable --now postgresql
```

### 建库与账号

```bash
sudo -u postgres psql
```

```sql
CREATE USER cuetiy WITH PASSWORD '改成强密码';
CREATE DATABASE cuetiy OWNER cuetiy ENCODING 'UTF8';
GRANT ALL PRIVILEGES ON DATABASE cuetiy TO cuetiy;
\q
```

远程连接时，改监听与白名单（路径因发行版而异）：

```text
# postgresql.conf
listen_addresses = '*'

# pg_hba.conf  （仅放行你的应用服务器 IP，不要 0.0.0.0/0 裸奔）
host  cuetiy  cuetiy  <应用服务器IP>/32  scram-sha-256
```

```bash
sudo systemctl restart postgresql
```

防火墙放行 `5432`（或你改过的端口），建议只对应用机开放。

---

## 2. 后端配置

工作目录一般是 `backend/output`（exe 与 `.env` 同目录）。

```env
DB_DRIVER=postgres
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=cuetiy
DB_PASSWORD=改成强密码
DB_NAME=cuetiy
DB_SSLMODE=disable

SERVER_PORT=8080
JWT_SECRET=改成随机长串
DEEPSEEK_API_KEY=...
STORAGE_DIR=./data/files
ARCHIVE_DIR=./data/archives
SKILLS_DIR=./skills
```

| 场景 | 补充 |
|------|------|
| 后端与 PG 同机 | `DB_HOST=127.0.0.1`，`DB_SSLMODE=disable` 即可 |
| PG 在另一台机 | `DB_HOST=<pg内网IP>`，`pg_hba.conf` 放行应用机，建议 `DB_SSLMODE=require` |
| 仍用 MySQL | `DB_DRIVER=mysql`，`DB_PORT=3306`，其余不变 |
| 本地 EXE 离线包 | `DB_DRIVER=sqlite`，`SQLITE_PATH=./data/cuetiy.db`（可省略） |

表结构 **以启动时 GORM AutoMigrate 为准**。手动兜底脚本：

- PostgreSQL：`backend/sql/schema.pg.sql`
- MySQL：`backend/sql/schema.sql`

启动：

```bash
cd backend/output   # 或你的部署目录
./cuetiy-backend    # Windows: cuetiy-backend.exe
```

日志应出现：`DB driver=postgres target=...`

健康检查（可选）：

```bash
curl -s http://127.0.0.1:8080/api/auth/login -X POST \
  -H 'Content-Type: application/json' \
  -d '{"email":"not-exist@x.com","password":"x"}'
# 能返回 JSON 即 HTTP 正常；库连不上会在启动日志 panic
```

---

## 3. （可选）MySQL → PostgreSQL 数据迁移

**先**在 PG 上用新后端空库启动一次，让 AutoMigrate 建好表，再灌数据。

### 方式 A：pgLoader（推荐）

```bash
# Debian: sudo apt install pgloader
pgloader mysql://root:MYSQL密码@MYSQL主机/cuetiy \
         postgresql://cuetiy:PG密码@PG主机/cuetiy
```

注意：

1. 迁移前停写 MySQL 后端（或只读窗口）。
2. bool 字段、时间字段 pgLoader 一般会自动映射；完成后抽查 `users.email`、`messages.content`（中文）、`personas`。
3. 迁完后 **核对自增序列**（否则新插入可能主键冲突）：

```sql
-- 在 cuetiy 库执行，表名按实际（GORM 为复数 snake_case）
SELECT setval(pg_get_serial_sequence('users','id'), COALESCE((SELECT MAX(id) FROM users), 0)+1, false);
SELECT setval(pg_get_serial_sequence('conversations','id'), COALESCE((SELECT MAX(id) FROM conversations), 0)+1, false);
SELECT setval(pg_get_serial_sequence('messages','id'), COALESCE((SELECT MAX(id) FROM messages), 0)+1, false);
SELECT setval(pg_get_serial_sequence('personas','id'), COALESCE((SELECT MAX(id) FROM personas), 0)+1, false);
SELECT setval(pg_get_serial_sequence('persona_files','id'), COALESCE((SELECT MAX(id) FROM persona_files), 0)+1, false);
SELECT setval(pg_get_serial_sequence('file_records','id'), COALESCE((SELECT MAX(id) FROM file_records), 0)+1, false);
SELECT setval(pg_get_serial_sequence('conversation_summaries','id'), COALESCE((SELECT MAX(id) FROM conversation_summaries), 0)+1, false);
SELECT setval(pg_get_serial_sequence('conversation_memories','id'), COALESCE((SELECT MAX(id) FROM conversation_memories), 0)+1, false);
SELECT setval(pg_get_serial_sequence('conversation_skill_states','id'), COALESCE((SELECT MAX(id) FROM conversation_skill_states), 0)+1, false);
SELECT setval(pg_get_serial_sequence('chat_archives','id'), COALESCE((SELECT MAX(id) FROM chat_archives), 0)+1, false);
SELECT setval(pg_get_serial_sequence('user_voices','id'), COALESCE((SELECT MAX(id) FROM user_voices), 0)+1, false);
```

### 方式 B：按表顺序导出/导入（无 pgLoader 时）

示例（金额小、结构简单时可用；生产更推荐 A）：

```bash
# MySQL 导出（只导数据，不导建表）
mysqldump -u root -p --no-create-info --complete-insert --skip-extended-insert \
  cuetiy users conversations messages personas persona_files file_records \
  conversation_summaries conversation_memories conversation_skill_states \
  chat_archives user_voices > cuetiy_data.sql
```

再手工把 `INSERT` 转成 PG 可执行语句（bool 的 0/1、时间格式），或写小脚本按 GORM 模型读 MySQL、写 PG。  
**文件资源不走 SQL**：把服务器上的 `STORAGE_DIR`、`ARCHIVE_DIR`、`skills/` 整目录一并打包到新机器相同相对路径。

### 迁移后检查清单

- [ ] 登录旧账号成功  
- [ ] 会话列表与历史消息完整  
- [ ] 头像 / 图片 / 语音能打开（`/storage/...`）  
- [ ] 人格与技能仍在  
- [ ] 新发一条消息能落库（主键序列已 setval）  
- [ ] 后端 `DB_DRIVER=postgres` 启动无 panic  

---

## 4. 与客户端包的关系

| 包 | 数据位置 | 配置 |
|----|----------|------|
| 云端 Web / thin 客户端 | 服务器 PG | App 内设置 API 地址指向该服务器 |
| EXE-unified 离线 | 本机 SQLite + `data/` | `DB_DRIVER=sqlite` |
| EXE/APK thin | 服务器 PG | 运行时配置 API URL（不打进包里） |

本地 ↔ 云端搬迁：使用 App 内 **聊天导出/导入 JSON**（`cuetiy-chat-export` v1），不依赖库引擎。

---

## 5. 回滚

1. 保留 MySQL 服务与数据目录直至 PG 稳定运行。  
2. `.env` 改回 `DB_DRIVER=mysql` 即可切回旧库。  
3. 两边不要同时对同一业务写入（没有双写同步）。
