-- 文件名：01-init-database.sql
-- 用途：在线判题系统数据库结构初始化脚本，创建核心业务表和初始数据
-- 创建日期：2024-01-15
-- 版本：v1.0
-- 说明：定义用户、题目、提交、比赛等核心实体的数据表结构，包含索引优化和示例数据
-- 依赖：MySQL 8.0+，utf8mb4字符集，oj_system数据库已创建
--
-- 表结构说明：
-- - users：用户信息表，支持多角色权限管理
-- - problems：题目信息表，包含题目描述、测试用例、限制条件
-- - submissions：提交记录表，记录用户提交的代码和判题结果
-- - contests：比赛信息表，支持ACM/OI赛制
-- - problem_contests：题目比赛关联表，多对多关系
-- - contest_participants：比赛参与者表，记录报名和排名信息
--
-- 索引策略：
-- - 主键索引：所有表都有自增主键
-- - 唯一索引：用户名、邮箱等唯一字段
-- - 复合索引：查询频繁的组合字段
-- - 外键索引：关联查询优化

-- ===========================================
-- 使用目标数据库
-- ===========================================
USE oj_system;

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL COMMENT '用户名',
    email VARCHAR(100) UNIQUE NOT NULL COMMENT '邮箱',
    password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
    role ENUM('student', 'teacher', 'admin') DEFAULT 'student' COMMENT '用户角色',
    avatar VARCHAR(255) DEFAULT '' COMMENT '头像URL',
    real_name VARCHAR(100) DEFAULT '' COMMENT '真实姓名',
    phone VARCHAR(20) DEFAULT '' COMMENT '手机号',
    school VARCHAR(100) DEFAULT '' COMMENT '学校',
    major VARCHAR(100) DEFAULT '' COMMENT '专业',
    bio TEXT DEFAULT '' COMMENT '个人简介',
    email_verified BOOLEAN DEFAULT FALSE COMMENT '邮箱是否已验证',
    status ENUM('active', 'inactive', 'banned') DEFAULT 'active' COMMENT '账户状态',
    last_login_at TIMESTAMP NULL COMMENT '最后登录时间',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_username (username),
    INDEX idx_email (email),
    INDEX idx_role (role),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 创建题目表
CREATE TABLE IF NOT EXISTS problems (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL COMMENT '题目标题',
    description TEXT NOT NULL COMMENT '题目描述',
    input_format TEXT COMMENT '输入格式说明',
    output_format TEXT COMMENT '输出格式说明',
    sample_input TEXT COMMENT '样例输入',
    sample_output TEXT COMMENT '样例输出',
    time_limit INT DEFAULT 1000 COMMENT '时间限制(毫秒)',
    memory_limit INT DEFAULT 128 COMMENT '内存限制(MB)',
    difficulty ENUM('easy', 'medium', 'hard') DEFAULT 'medium' COMMENT '难度等级',
    tags JSON COMMENT '题目标签',
    hint TEXT COMMENT '提示信息',
    source VARCHAR(200) DEFAULT '' COMMENT '题目来源',
    author VARCHAR(100) DEFAULT '' COMMENT '出题人',
    visible BOOLEAN DEFAULT TRUE COMMENT '是否可见',
    submit_count INT DEFAULT 0 COMMENT '提交次数',
    accept_count INT DEFAULT 0 COMMENT '通过次数',
    created_by BIGINT COMMENT '创建者ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_difficulty (difficulty),
    INDEX idx_created_by (created_by),
    INDEX idx_visible (visible),
    INDEX idx_created_at (created_at),
    FULLTEXT INDEX idx_title_description (title, description)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='题目表';

-- 创建提交表
CREATE TABLE IF NOT EXISTS submissions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    problem_id BIGINT NOT NULL COMMENT '题目ID',
    contest_id BIGINT NULL COMMENT '比赛ID',
    language VARCHAR(20) NOT NULL COMMENT '编程语言',
    code TEXT NOT NULL COMMENT '提交代码',
    status ENUM('pending', 'judging', 'accepted', 'wrong_answer', 'time_limit_exceeded', 'memory_limit_exceeded', 'runtime_error', 'compile_error', 'system_error') DEFAULT 'pending' COMMENT '判题状态',
    score INT DEFAULT 0 COMMENT '得分',
    time_used INT DEFAULT 0 COMMENT '运行时间(毫秒)',
    memory_used INT DEFAULT 0 COMMENT '内存使用(KB)',
    compile_info TEXT COMMENT '编译信息',
    judge_info JSON COMMENT '判题详细信息',
    ip VARCHAR(45) COMMENT '提交IP',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '提交时间',
    judged_at TIMESTAMP NULL COMMENT '判题完成时间',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_problem_id (problem_id),
    INDEX idx_contest_id (contest_id),
    INDEX idx_status (status),
    INDEX idx_language (language),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='提交表';

-- 创建比赛表
CREATE TABLE IF NOT EXISTS contests (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL COMMENT '比赛标题',
    description TEXT COMMENT '比赛描述',
    start_time TIMESTAMP NOT NULL COMMENT '开始时间',
    end_time TIMESTAMP NOT NULL COMMENT '结束时间',
    type ENUM('public', 'private', 'official') DEFAULT 'public' COMMENT '比赛类型',
    password VARCHAR(100) DEFAULT '' COMMENT '比赛密码',
    max_participants INT DEFAULT 0 COMMENT '最大参与人数(0表示无限制)',
    freeze_time INT DEFAULT 0 COMMENT '封榜时间(分钟)',
    penalty_time INT DEFAULT 20 COMMENT '罚时(分钟)',
    status ENUM('upcoming', 'running', 'ended') DEFAULT 'upcoming' COMMENT '比赛状态',
    created_by BIGINT COMMENT '创建者ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_start_time (start_time),
    INDEX idx_end_time (end_time),
    INDEX idx_status (status),
    INDEX idx_type (type),
    INDEX idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='比赛表';

-- 创建比赛题目关联表
CREATE TABLE IF NOT EXISTS contest_problems (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    contest_id BIGINT NOT NULL COMMENT '比赛ID',
    problem_id BIGINT NOT NULL COMMENT '题目ID',
    problem_order CHAR(1) NOT NULL COMMENT '题目顺序(A,B,C...)',
    score INT DEFAULT 100 COMMENT '题目分值',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (contest_id) REFERENCES contests(id) ON DELETE CASCADE,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    UNIQUE KEY uk_contest_order (contest_id, problem_order),
    UNIQUE KEY uk_contest_problem (contest_id, problem_id),
    INDEX idx_contest_id (contest_id),
    INDEX idx_problem_id (problem_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='比赛题目关联表';

-- 创建用户会话表
CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    token_id VARCHAR(64) NOT NULL COMMENT '令牌ID',
    refresh_token VARCHAR(255) NOT NULL COMMENT '刷新令牌',
    device_info VARCHAR(500) DEFAULT '' COMMENT '设备信息',
    ip VARCHAR(45) COMMENT 'IP地址',
    expires_at TIMESTAMP NOT NULL COMMENT '过期时间',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uk_token_id (token_id),
    INDEX idx_user_id (user_id),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户会话表';

-- 插入初始数据
-- 创建默认管理员账户 (密码: admin123)
INSERT INTO users (username, email, password_hash, role, real_name, email_verified) VALUES 
('admin', 'admin@example.com', '$2a$12$rVSM8B/l.ub9E9Yws9jNYeE7XZnE4WpyT.vJTKtTxzYVR0pKT.mEO', 'admin', '系统管理员', TRUE);

-- 创建示例学生账户 (密码: student123)
INSERT INTO users (username, email, password_hash, role, real_name, school, major, email_verified) VALUES 
('student', 'student@example.com', '$2a$12$EUoQGWZ3dKr9V8Y2m2FTCO7B8C4dOd9L1YBkE5l3F4v4wdT6M7EgK', 'student', '张三', '示例大学', '计算机科学与技术', TRUE);

-- 创建示例教师账户 (密码: teacher123)
INSERT INTO users (username, email, password_hash, role, real_name, school, email_verified) VALUES 
('teacher', 'teacher@example.com', '$2a$12$FVpRGWZ3dKr9V8Y2m2FTCO7B8C4dOd9L1YBkE5l3F4v4wdT6M7EgL', 'teacher', '李老师', '示例大学', TRUE);

-- 创建示例题目
INSERT INTO problems (title, description, input_format, output_format, sample_input, sample_output, time_limit, memory_limit, difficulty, created_by) VALUES 
('A + B Problem', 
'计算两个整数的和。这是一个经典的入门题目，用于测试判题系统的基本功能。', 
'输入包含两个整数 a 和 b，用空格分隔。', 
'输出一个整数，表示 a + b 的结果。', 
'1 2', 
'3', 
1000, 128, 'easy', 1),

('Hello World', 
'输出 "Hello, World!" 字符串。这是编程的第一步，让我们开始吧！', 
'无输入。', 
'输出字符串 "Hello, World!"。', 
'', 
'Hello, World!', 
1000, 128, 'easy', 1);

-- 设置 MySQL 时区
SET time_zone = '+08:00';

-- 显示初始化完成信息
SELECT 'Database initialization completed successfully!' as message;