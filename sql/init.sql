-- 在线判题系统数据库初始化脚本
-- 创建日期：2024-01-15
-- 版本：v1.0
-- 说明：创建所有必需的表结构和初始数据

-- ===========================================
-- 创建数据库和用户
-- ===========================================

-- 创建主数据库
CREATE DATABASE IF NOT EXISTS oj_system 
  CHARACTER SET utf8mb4 
  COLLATE utf8mb4_unicode_ci;

-- 创建用户服务专用数据库
CREATE DATABASE IF NOT EXISTS oj_users 
  CHARACTER SET utf8mb4 
  COLLATE utf8mb4_unicode_ci;

-- 创建数据库用户
CREATE USER IF NOT EXISTS 'oj_user'@'%' IDENTIFIED BY 'oj_password';
GRANT ALL PRIVILEGES ON oj_system.* TO 'oj_user'@'%';
GRANT ALL PRIVILEGES ON oj_users.* TO 'oj_user'@'%';
FLUSH PRIVILEGES;

-- 使用用户数据库
USE oj_users;

-- ===========================================
-- 用户模块数据表
-- ===========================================

-- 用户基础信息表
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '用户唯一标识',
    username VARCHAR(50) UNIQUE NOT NULL COMMENT '用户名，全局唯一',
    email VARCHAR(100) UNIQUE NOT NULL COMMENT '邮箱地址，用于登录和通知',
    password_hash VARCHAR(255) NOT NULL COMMENT 'bcrypt加密后的密码哈希',
    real_name VARCHAR(100) DEFAULT '' COMMENT '真实姓名',
    avatar_url VARCHAR(500) DEFAULT '' COMMENT '头像链接',
    bio TEXT COMMENT '个人简介',
    role ENUM('student', 'teacher', 'admin') DEFAULT 'student' COMMENT '用户角色',
    status ENUM('active', 'inactive', 'banned') DEFAULT 'active' COMMENT '账户状态',
    email_verified BOOLEAN DEFAULT FALSE COMMENT '邮箱是否已验证',
    last_login_at TIMESTAMP NULL COMMENT '最后登录时间',
    last_login_ip VARCHAR(45) DEFAULT '' COMMENT '最后登录IP地址',
    login_count INT DEFAULT 0 COMMENT '累计登录次数',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    -- 索引设计
    INDEX idx_username (username) COMMENT '用户名查询索引',
    INDEX idx_email (email) COMMENT '邮箱查询索引',
    INDEX idx_role (role) COMMENT '角色筛选索引',
    INDEX idx_status (status) COMMENT '状态筛选索引',
    INDEX idx_created_at (created_at) COMMENT '创建时间排序索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户基础信息表';

-- 用户令牌表
CREATE TABLE user_tokens (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '令牌记录唯一标识',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    token_id VARCHAR(64) UNIQUE NOT NULL COMMENT 'JWT令牌唯一标识(jti)',
    refresh_token VARCHAR(500) NOT NULL COMMENT '刷新令牌',
    access_token_expire TIMESTAMP NOT NULL COMMENT '访问令牌过期时间',
    refresh_token_expire TIMESTAMP NOT NULL COMMENT '刷新令牌过期时间',
    client_info VARCHAR(1000) DEFAULT '' COMMENT '客户端信息(格式化字符串: user_agent|ip_address)',
    is_revoked BOOLEAN DEFAULT FALSE COMMENT '是否已撤销',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- 索引设计
    INDEX idx_user_id (user_id) COMMENT '用户ID查询索引',
    INDEX idx_token_id (token_id) COMMENT '令牌ID查询索引',
    INDEX idx_refresh_expire (refresh_token_expire) COMMENT '令牌过期时间索引',
    INDEX idx_is_revoked (is_revoked) COMMENT '撤销状态索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户令牌管理表';

-- 用户登录日志表
CREATE TABLE user_login_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '日志记录唯一标识',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    login_type ENUM('password', 'refresh_token', 'oauth') DEFAULT 'password' COMMENT '登录方式',
    ip_address VARCHAR(45) NOT NULL COMMENT '登录IP地址',
    user_agent TEXT COMMENT '浏览器用户代理信息',
    login_status ENUM('success', 'failed', 'blocked') NOT NULL COMMENT '登录状态',
    failure_reason VARCHAR(200) DEFAULT '' COMMENT '登录失败原因',
    location_info VARCHAR(500) DEFAULT '' COMMENT 'IP地理位置信息(格式化字符串: 国家|省份|城市)',
    device_info VARCHAR(500) DEFAULT '' COMMENT '设备信息(格式化字符串: 操作系统|浏览器|设备类型)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- 索引设计
    INDEX idx_user_id (user_id) COMMENT '用户ID查询索引',
    INDEX idx_ip_address (ip_address) COMMENT 'IP地址查询索引',
    INDEX idx_login_status (login_status) COMMENT '登录状态查询索引',
    INDEX idx_created_at (created_at) COMMENT '登录时间排序索引',
    INDEX idx_user_time (user_id, created_at) COMMENT '用户登录时间复合索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户登录日志表';

-- 用户统计信息表
CREATE TABLE user_statistics (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '统计记录唯一标识',
    user_id BIGINT UNIQUE NOT NULL COMMENT '用户ID',
    
    -- 提交统计
    total_submissions INT DEFAULT 0 COMMENT '总提交次数',
    accepted_submissions INT DEFAULT 0 COMMENT '通过的提交次数',
    
    -- 题目统计
    solved_problems INT DEFAULT 0 COMMENT '已解决题目数量',
    easy_solved INT DEFAULT 0 COMMENT '简单题目解决数量',
    medium_solved INT DEFAULT 0 COMMENT '中等题目解决数量',
    hard_solved INT DEFAULT 0 COMMENT '困难题目解决数量',
    
    -- 排名信息
    current_rating INT DEFAULT 1200 COMMENT '当前评分',
    max_rating INT DEFAULT 1200 COMMENT '历史最高评分',
    rank_level ENUM('bronze', 'silver', 'gold', 'platinum', 'diamond') DEFAULT 'bronze' COMMENT '段位等级',
    
    -- 时间统计
    total_code_time INT DEFAULT 0 COMMENT '总编程时间(分钟)',
    average_solve_time INT DEFAULT 0 COMMENT '平均解题时间(分钟)',
    
    -- 比赛统计
    contest_participated INT DEFAULT 0 COMMENT '参与比赛次数',
    contest_won INT DEFAULT 0 COMMENT '比赛获胜次数',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- 索引设计
    INDEX idx_user_id (user_id) COMMENT '用户ID查询索引',
    INDEX idx_solved_problems (solved_problems) COMMENT '解题数量排序索引',
    INDEX idx_current_rating (current_rating) COMMENT '评分排序索引',
    INDEX idx_rank_level (rank_level) COMMENT '段位等级查询索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户统计信息表';

-- 使用主数据库
USE oj_system;

-- ===========================================
-- 题目模块数据表
-- ===========================================

-- 题目基础信息表
CREATE TABLE problems (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '题目唯一标识',
    title VARCHAR(200) NOT NULL COMMENT '题目标题',
    description TEXT NOT NULL COMMENT '题目描述',
    input_format TEXT COMMENT '输入格式说明',
    output_format TEXT COMMENT '输出格式说明',
    sample_input TEXT COMMENT '示例输入',
    sample_output TEXT COMMENT '示例输出',
    hint TEXT COMMENT '题目提示',
    
    -- 限制参数
    time_limit INT DEFAULT 1000 COMMENT '时间限制(毫秒)',
    memory_limit INT DEFAULT 128 COMMENT '内存限制(MB)',
    
    -- 分类信息
    difficulty ENUM('easy', 'medium', 'hard') DEFAULT 'medium' COMMENT '难度等级',
    category VARCHAR(50) DEFAULT '' COMMENT '题目分类',
    tags JSON COMMENT '题目标签',
    
    -- 统计信息
    total_submissions INT DEFAULT 0 COMMENT '总提交次数',
    accepted_submissions INT DEFAULT 0 COMMENT '通过次数',
    
    -- 状态管理
    status ENUM('draft', 'published', 'hidden') DEFAULT 'draft' COMMENT '题目状态',
    created_by BIGINT NOT NULL COMMENT '创建者用户ID',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    -- 索引设计
    INDEX idx_difficulty (difficulty) COMMENT '难度查询索引',
    INDEX idx_category (category) COMMENT '分类查询索引',
    INDEX idx_status (status) COMMENT '状态查询索引',
    INDEX idx_created_by (created_by) COMMENT '创建者查询索引',
    INDEX idx_created_at (created_at) COMMENT '创建时间排序索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='题目基础信息表';

-- 测试用例表
CREATE TABLE test_cases (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '测试用例唯一标识',
    problem_id BIGINT NOT NULL COMMENT '题目ID',
    input_data TEXT NOT NULL COMMENT '输入数据',
    expected_output TEXT NOT NULL COMMENT '期望输出',
    is_sample BOOLEAN DEFAULT FALSE COMMENT '是否为示例用例',
    score INT DEFAULT 10 COMMENT '测试用例分值',
    sort_order INT DEFAULT 0 COMMENT '排序顺序',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    
    -- 外键约束
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    
    -- 索引设计
    INDEX idx_problem_id (problem_id) COMMENT '题目ID查询索引',
    INDEX idx_is_sample (is_sample) COMMENT '示例用例查询索引',
    INDEX idx_sort_order (sort_order) COMMENT '排序查询索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='测试用例表';

-- ===========================================
-- 提交模块数据表
-- ===========================================

-- 代码提交表
CREATE TABLE submissions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '提交记录唯一标识',
    user_id BIGINT NOT NULL COMMENT '提交用户ID',
    problem_id BIGINT NOT NULL COMMENT '题目ID',
    contest_id BIGINT DEFAULT NULL COMMENT '比赛ID(如果是比赛提交)',
    
    -- 代码信息
    language VARCHAR(20) NOT NULL COMMENT '编程语言',
    code TEXT NOT NULL COMMENT '提交的代码',
    code_length INT DEFAULT 0 COMMENT '代码长度(字符数)',
    
    -- 判题结果
    status ENUM('pending', 'judging', 'accepted', 'wrong_answer', 'time_limit_exceeded', 
                'memory_limit_exceeded', 'runtime_error', 'compile_error', 'system_error') 
           DEFAULT 'pending' COMMENT '判题状态',
    
    -- 执行信息
    time_used INT DEFAULT 0 COMMENT '执行时间(毫秒)',
    memory_used INT DEFAULT 0 COMMENT '内存使用(KB)',
    score INT DEFAULT 0 COMMENT '得分',
    
    -- 判题详情
    compile_info TEXT COMMENT '编译信息',
    runtime_info TEXT COMMENT '运行时信息',
    test_case_results JSON COMMENT '测试用例执行结果',
    
    -- 系统信息
    judge_server VARCHAR(100) DEFAULT '' COMMENT '判题服务器',
    ip_address VARCHAR(45) DEFAULT '' COMMENT '提交IP地址',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '提交时间',
    judged_at TIMESTAMP NULL COMMENT '判题完成时间',
    
    -- 索引设计
    INDEX idx_user_id (user_id) COMMENT '用户ID查询索引',
    INDEX idx_problem_id (problem_id) COMMENT '题目ID查询索引',
    INDEX idx_contest_id (contest_id) COMMENT '比赛ID查询索引',
    INDEX idx_status (status) COMMENT '状态查询索引',
    INDEX idx_language (language) COMMENT '语言查询索引',
    INDEX idx_created_at (created_at) COMMENT '提交时间排序索引',
    INDEX idx_user_problem (user_id, problem_id) COMMENT '用户题目复合索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代码提交表';

-- ===========================================
-- 比赛模块数据表
-- ===========================================

-- 比赛信息表
CREATE TABLE contests (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '比赛唯一标识',
    title VARCHAR(200) NOT NULL COMMENT '比赛标题',
    description TEXT COMMENT '比赛描述',
    
    -- 时间管理
    start_time TIMESTAMP NOT NULL COMMENT '开始时间',
    end_time TIMESTAMP NOT NULL COMMENT '结束时间',
    duration INT NOT NULL COMMENT '比赛时长(分钟)',
    
    -- 比赛类型
    type ENUM('acm', 'oi', 'practice') DEFAULT 'acm' COMMENT '比赛类型',
    status ENUM('upcoming', 'running', 'ended') DEFAULT 'upcoming' COMMENT '比赛状态',
    
    -- 权限设置
    is_public BOOLEAN DEFAULT TRUE COMMENT '是否公开比赛',
    password VARCHAR(100) DEFAULT '' COMMENT '比赛密码',
    max_participants INT DEFAULT 0 COMMENT '最大参与人数(0表示不限制)',
    
    -- 排名设置
    freeze_time INT DEFAULT 60 COMMENT '封榜时间(分钟)',
    unfreeze_time TIMESTAMP NULL COMMENT '解封时间',
    
    created_by BIGINT NOT NULL COMMENT '创建者用户ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    -- 索引设计
    INDEX idx_start_time (start_time) COMMENT '开始时间查询索引',
    INDEX idx_status (status) COMMENT '状态查询索引',
    INDEX idx_type (type) COMMENT '类型查询索引',
    INDEX idx_created_by (created_by) COMMENT '创建者查询索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='比赛信息表';

-- ===========================================
-- 系统模块数据表
-- ===========================================

-- 系统配置表
CREATE TABLE system_configs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '配置项唯一标识',
    config_key VARCHAR(100) UNIQUE NOT NULL COMMENT '配置键名',
    config_value TEXT COMMENT '配置值',
    config_type ENUM('string', 'number', 'boolean', 'json') DEFAULT 'string' COMMENT '配置类型',
    description VARCHAR(500) DEFAULT '' COMMENT '配置描述',
    is_public BOOLEAN DEFAULT FALSE COMMENT '是否公开配置',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_config_key (config_key) COMMENT '配置键查询索引',
    INDEX idx_is_public (is_public) COMMENT '公开配置查询索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表';

-- ===========================================
-- 初始化数据
-- ===========================================

-- 插入初始系统配置
INSERT INTO system_configs (config_key, config_value, config_type, description, is_public) VALUES
('site_name', '在线判题系统', 'string', '网站名称', true),
('site_description', '基于Go语言的高性能在线判题平台', 'string', '网站描述', true),
('max_submission_size', '65536', 'number', '最大代码提交大小(字节)', false),
('supported_languages', '["cpp", "java", "python", "go", "javascript"]', 'json', '支持的编程语言', true),
('default_time_limit', '1000', 'number', '默认时间限制(毫秒)', false),
('default_memory_limit', '128', 'number', '默认内存限制(MB)', false),
('registration_enabled', 'true', 'boolean', '是否允许用户注册', true),
('email_verification_required', 'false', 'boolean', '是否需要邮箱验证', false);

-- 切换到用户数据库插入初始用户
USE oj_users;

-- 插入默认管理员用户 (密码: admin123)
-- bcrypt hash for "admin123" with cost 12
INSERT INTO users (username, email, password_hash, real_name, role, status, email_verified) VALUES
('admin', 'admin@oj.local', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8.1a4J9YzL9k2ZJm3Kq', '系统管理员', 'admin', 'active', true);

-- 初始化管理员统计信息
INSERT INTO user_statistics (user_id) VALUES (1);

-- 显示创建完成信息
SELECT 'Database initialization completed successfully!' as status;