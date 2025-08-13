-- 题目服务数据库初始化脚本
-- 创建日期：2024-01-15
-- 版本：v1.0
-- 说明：创建题目管理相关的表结构

-- ===========================================
-- 创建题目服务专用数据库
-- ===========================================

-- 创建题目服务数据库
CREATE DATABASE IF NOT EXISTS oj_problems 
  CHARACTER SET utf8mb4 
  COLLATE utf8mb4_unicode_ci;

-- 授权数据库访问权限
GRANT ALL PRIVILEGES ON oj_problems.* TO 'oj_user'@'%';
FLUSH PRIVILEGES;

-- 使用题目数据库
USE oj_problems;

-- ===========================================
-- 题目管理数据表
-- ===========================================

-- 题目基础信息表
CREATE TABLE problems (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '题目唯一标识',
    title VARCHAR(200) NOT NULL COMMENT '题目标题',
    description TEXT NOT NULL COMMENT '题目描述内容',
    input_format TEXT COMMENT '输入格式说明',
    output_format TEXT COMMENT '输出格式说明',
    sample_input TEXT COMMENT '样例输入',
    sample_output TEXT COMMENT '样例输出',
    difficulty ENUM('easy', 'medium', 'hard') DEFAULT 'medium' COMMENT '难度等级',
    time_limit INT DEFAULT 1000 COMMENT '时间限制（毫秒）',
    memory_limit INT DEFAULT 128 COMMENT '内存限制（MB）',
    languages JSON COMMENT '支持的编程语言列表',
    tags JSON COMMENT '题目标签列表',
    created_by BIGINT NOT NULL COMMENT '创建者用户ID',
    is_public BOOLEAN DEFAULT TRUE COMMENT '是否公开',
    submission_count INT DEFAULT 0 COMMENT '总提交次数',
    accepted_count INT DEFAULT 0 COMMENT '通过次数',
    acceptance_rate DECIMAL(5,2) DEFAULT 0.00 COMMENT '通过率百分比',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL COMMENT '软删除时间',
    
    -- 索引设计
    INDEX idx_difficulty (difficulty) COMMENT '难度查询索引',
    INDEX idx_created_by (created_by) COMMENT '创建者查询索引',
    INDEX idx_created_at (created_at) COMMENT '创建时间排序索引',
    INDEX idx_acceptance_rate (acceptance_rate) COMMENT '通过率排序索引',
    INDEX idx_public_status (is_public, deleted_at) COMMENT '公开状态复合索引',
    INDEX idx_submission_count (submission_count) COMMENT '提交次数排序索引',
    FULLTEXT INDEX idx_title_description (title, description) COMMENT '全文搜索索引'
) ENGINE=InnoDB 
  DEFAULT CHARSET=utf8mb4 
  COLLATE=utf8mb4_unicode_ci 
  COMMENT='题目基础信息表';

-- 题目标签表（标准化标签管理）
CREATE TABLE problem_tags (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '标签唯一标识',
    name VARCHAR(50) NOT NULL UNIQUE COMMENT '标签名称',
    display_name VARCHAR(100) COMMENT '显示名称',
    description TEXT COMMENT '标签描述',
    color VARCHAR(7) DEFAULT '#007bff' COMMENT '标签颜色（十六进制）',
    usage_count INT DEFAULT 0 COMMENT '使用次数',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_usage_count (usage_count) COMMENT '使用次数索引'
) ENGINE=InnoDB 
  DEFAULT CHARSET=utf8mb4 
  COLLATE=utf8mb4_unicode_ci 
  COMMENT='题目标签表';

-- 题目-标签关联表
CREATE TABLE problem_tag_relations (
    problem_id BIGINT NOT NULL COMMENT '题目ID',
    tag_id BIGINT NOT NULL COMMENT '标签ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '关联创建时间',
    
    PRIMARY KEY (problem_id, tag_id),
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES problem_tags(id) ON DELETE CASCADE,
    INDEX idx_tag_id (tag_id) COMMENT '标签反向查询索引'
) ENGINE=InnoDB 
  DEFAULT CHARSET=utf8mb4 
  COLLATE=utf8mb4_unicode_ci 
  COMMENT='题目标签关联表';

-- 题目统计表（定期汇总统计数据）
CREATE TABLE problem_statistics (
    problem_id BIGINT PRIMARY KEY COMMENT '题目ID',
    total_submissions INT DEFAULT 0 COMMENT '总提交次数',
    accepted_submissions INT DEFAULT 0 COMMENT '通过次数',
    wrong_answer_count INT DEFAULT 0 COMMENT '答案错误次数',
    time_limit_exceeded_count INT DEFAULT 0 COMMENT '超时次数',
    memory_limit_exceeded_count INT DEFAULT 0 COMMENT '内存超限次数',
    runtime_error_count INT DEFAULT 0 COMMENT '运行时错误次数',
    compile_error_count INT DEFAULT 0 COMMENT '编译错误次数',
    acceptance_rate DECIMAL(5,2) DEFAULT 0.00 COMMENT '通过率',
    avg_runtime INT DEFAULT 0 COMMENT '平均运行时间（毫秒）',
    avg_memory INT DEFAULT 0 COMMENT '平均内存使用（KB）',
    last_submission_at TIMESTAMP NULL COMMENT '最后提交时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '统计更新时间',
    
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
) ENGINE=InnoDB 
  DEFAULT CHARSET=utf8mb4 
  COLLATE=utf8mb4_unicode_ci 
  COMMENT='题目统计信息表';

-- 题目版本历史表（用于内容变更追踪）
CREATE TABLE problem_versions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '版本记录ID',
    problem_id BIGINT NOT NULL COMMENT '题目ID',
    version_number VARCHAR(20) NOT NULL COMMENT '版本号',
    title VARCHAR(200) NOT NULL COMMENT '题目标题（历史版本）',
    description TEXT NOT NULL COMMENT '题目描述（历史版本）',
    change_log TEXT COMMENT '变更说明',
    changed_by BIGINT NOT NULL COMMENT '修改者用户ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '版本创建时间',
    
    UNIQUE KEY uk_problem_version (problem_id, version_number),
    INDEX idx_problem_id (problem_id) COMMENT '题目查询索引',
    INDEX idx_created_at (created_at) COMMENT '时间排序索引',
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
) ENGINE=InnoDB 
  DEFAULT CHARSET=utf8mb4 
  COLLATE=utf8mb4_unicode_ci 
  COMMENT='题目版本历史表';

-- ===========================================
-- 初始化基础数据
-- ===========================================

-- 插入基础标签数据
INSERT INTO problem_tags (name, display_name, description, color) VALUES
('array', '数组', '涉及数组操作和处理的题目', '#28a745'),
('string', '字符串', '字符串处理和操作相关题目', '#17a2b8'),
('math', '数学', '数学计算和算法相关题目', '#ffc107'),
('dp', '动态规划', '动态规划算法题目', '#dc3545'),
('greedy', '贪心算法', '贪心策略解决的题目', '#6f42c1'),
('graph', '图论', '图的遍历和算法题目', '#fd7e14'),
('tree', '树结构', '二叉树和树形结构题目', '#20c997'),
('sort', '排序', '排序算法相关题目', '#6c757d'),
('search', '搜索', '搜索算法（BFS/DFS）题目', '#e83e8c'),
('hash', '哈希表', '哈希表和映射相关题目', '#007bff');

-- 插入示例题目
INSERT INTO problems (
    title, description, input_format, output_format, 
    sample_input, sample_output, difficulty, time_limit, 
    memory_limit, languages, tags, created_by, is_public
) VALUES 
(
    '两数之和',
    '给定一个整数数组 nums 和一个整数目标值 target，请你在该数组中找出和为目标值 target 的那两个整数，并返回它们的数组下标。\n\n你可以假设每种输入只会对应一个答案。但是，数组中同一个元素在答案里不能重复出现。\n\n你可以按任意顺序返回答案。',
    '第一行包含一个整数 n，表示数组长度。\n第二行包含 n 个整数，表示数组 nums。\n第三行包含一个整数 target，表示目标值。',
    '输出两个整数，表示和为 target 的两个数的下标（从0开始），用空格分隔。',
    '4\n2 7 11 15\n9',
    '0 1',
    'easy',
    1000,
    128,
    JSON_ARRAY('cpp', 'java', 'python', 'go'),
    JSON_ARRAY('array', 'hash'),
    1,
    TRUE
),
(
    '最长回文子串',
    '给你一个字符串 s，找到 s 中最长的回文子串。\n\n回文串是正着读和反着读都一样的字符串。',
    '输入一行字符串 s，长度不超过 1000。',
    '输出最长回文子串。如果有多个答案，输出任意一个即可。',
    'babad',
    'bab',
    'medium',
    2000,
    256,
    JSON_ARRAY('cpp', 'java', 'python', 'go'),
    JSON_ARRAY('string', 'dp'),
    1,
    TRUE
);

-- 建立示例题目和标签的关联关系
INSERT INTO problem_tag_relations (problem_id, tag_id)
SELECT 1, id FROM problem_tags WHERE name IN ('array', 'hash')
UNION ALL
SELECT 2, id FROM problem_tags WHERE name IN ('string', 'dp');

-- 更新标签使用计数
UPDATE problem_tags SET usage_count = (
    SELECT COUNT(*) FROM problem_tag_relations WHERE tag_id = problem_tags.id
);