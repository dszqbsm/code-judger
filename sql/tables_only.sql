-- 使用题目数据库
USE oj_problems;

-- ===========================================
-- 题目主表
-- ===========================================
CREATE TABLE IF NOT EXISTS problems (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '题目ID',
  title VARCHAR(200) NOT NULL COMMENT '题目标题',
  description TEXT NOT NULL COMMENT '题目描述',
  input_format TEXT COMMENT '输入格式说明',
  output_format TEXT COMMENT '输出格式说明',
  sample_input TEXT COMMENT '样例输入',
  sample_output TEXT COMMENT '样例输出',
  difficulty ENUM('easy', 'medium', 'hard') DEFAULT 'medium' COMMENT '难度等级',
  time_limit INT DEFAULT 1000 COMMENT '时间限制(毫秒)',
  memory_limit INT DEFAULT 128 COMMENT '内存限制(MB)',
  languages JSON COMMENT '支持的编程语言(JSON格式)',
  tags JSON COMMENT '题目标签(JSON格式)',
  created_by BIGINT NOT NULL COMMENT '创建者用户ID',
  is_public BOOLEAN DEFAULT true COMMENT '是否公开',
  submission_count INT DEFAULT 0 COMMENT '提交次数',
  accepted_count INT DEFAULT 0 COMMENT '通过次数',
  acceptance_rate DECIMAL(5,2) DEFAULT 0.00 COMMENT '通过率(%)',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
  
  -- 索引定义
  INDEX idx_title (title),
  INDEX idx_difficulty (difficulty),
  INDEX idx_created_by (created_by),
  INDEX idx_is_public (is_public),
  INDEX idx_acceptance_rate (acceptance_rate),
  INDEX idx_created_at (created_at),
  INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='题目信息表';

-- ===========================================
-- 标签表
-- ===========================================
CREATE TABLE IF NOT EXISTS tags (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '标签ID',
  name VARCHAR(100) NOT NULL UNIQUE COMMENT '标签名称',
  name_en VARCHAR(100) COMMENT '英文标签名',
  parent_id BIGINT COMMENT '父标签ID',
  level INT DEFAULT 0 COMMENT '标签层级',
  weight DECIMAL(3,2) DEFAULT 1.0 COMMENT '标签权重',
  usage_count BIGINT DEFAULT 0 COMMENT '使用次数',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  
  INDEX idx_parent_id (parent_id),
  INDEX idx_level (level),
  FOREIGN KEY (parent_id) REFERENCES tags(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='标签表';

-- ===========================================
-- 题目标签关联表
-- ===========================================
CREATE TABLE IF NOT EXISTS problem_tags (
  problem_id BIGINT NOT NULL COMMENT '题目ID',
  tag_id BIGINT NOT NULL COMMENT '标签ID',
  weight DECIMAL(3,2) DEFAULT 1.0 COMMENT '关联权重',
  is_auto_generated BOOLEAN DEFAULT false COMMENT '是否自动生成',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '关联时间',
  
  PRIMARY KEY (problem_id, tag_id),
  FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='题目标签关联表';

-- ===========================================
-- 题目统计表
-- ===========================================
CREATE TABLE IF NOT EXISTS problem_statistics (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '统计ID',
  problem_id BIGINT NOT NULL UNIQUE COMMENT '题目ID',
  total_submissions BIGINT DEFAULT 0 COMMENT '总提交数',
  accepted_submissions BIGINT DEFAULT 0 COMMENT '通过提交数',
  wrong_answer_count BIGINT DEFAULT 0 COMMENT '答案错误数',
  time_limit_exceeded_count BIGINT DEFAULT 0 COMMENT '超时数',
  memory_limit_exceeded_count BIGINT DEFAULT 0 COMMENT '内存超限数',
  runtime_error_count BIGINT DEFAULT 0 COMMENT '运行错误数',
  compile_error_count BIGINT DEFAULT 0 COMMENT '编译错误数',
  avg_time_used DECIMAL(8,2) DEFAULT 0 COMMENT '平均运行时间(ms)',
  avg_memory_used DECIMAL(8,2) DEFAULT 0 COMMENT '平均内存使用(KB)',
  first_accepted_at TIMESTAMP NULL COMMENT '首次通过时间',
  last_submitted_at TIMESTAMP NULL COMMENT '最后提交时间',
  
  FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='题目统计表';

-- ===========================================
-- 题目版本历史表
-- ===========================================
CREATE TABLE IF NOT EXISTS problem_versions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '版本ID',
  problem_id BIGINT NOT NULL COMMENT '题目ID',
  version_number INT NOT NULL COMMENT '版本号',
  title VARCHAR(200) NOT NULL COMMENT '题目标题',
  description TEXT NOT NULL COMMENT '题目描述',
  change_summary TEXT COMMENT '变更摘要',
  changed_by BIGINT NOT NULL COMMENT '变更人ID',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  
  INDEX idx_problem_id (problem_id),
  INDEX idx_version_number (version_number),
  UNIQUE KEY uk_problem_version (problem_id, version_number),
  FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='题目版本历史表';

-- ===========================================
-- 插入基础数据
-- ===========================================

-- 插入基础标签
INSERT IGNORE INTO tags (name, name_en, level, usage_count) VALUES
('数组', 'array', 1, 0),
('链表', 'linked-list', 1, 0),
('栈', 'stack', 1, 0),
('队列', 'queue', 1, 0),
('哈希表', 'hash-table', 1, 0),
('树', 'tree', 1, 0),
('图', 'graph', 1, 0),
('动态规划', 'dynamic-programming', 1, 0),
('贪心', 'greedy', 1, 0),
('回溯', 'backtracking', 1, 0),
('分治', 'divide-and-conquer', 1, 0),
('双指针', 'two-pointers', 1, 0),
('滑动窗口', 'sliding-window', 1, 0),
('排序', 'sorting', 1, 0),
('搜索', 'searching', 1, 0);

-- 插入示例题目
INSERT IGNORE INTO problems (
  title, description, input_format, output_format, 
  sample_input, sample_output, difficulty, time_limit, memory_limit, 
  languages, tags, created_by, is_public
) VALUES (
  '两数之和',
  '给定一个整数数组nums和一个整数目标值target，请你在该数组中找出和为目标值target的那两个整数，并返回它们的数组下标。',
  '第一行包含一个整数n，表示数组长度。第二行包含n个整数，表示数组nums。第三行包含一个整数target，表示目标值。',
  '输出两个整数，表示和为target的两个数的下标（从0开始），用空格分隔。',
  '4\n2 7 11 15\n9',
  '0 1',
  'easy',
  1000,
  128,
  '["cpp", "java", "python", "go"]',
  '["数组", "哈希表"]',
  1,
  true
);

-- 验证数据插入
SELECT 'Database initialization completed' AS status;