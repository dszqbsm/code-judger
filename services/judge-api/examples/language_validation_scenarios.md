# 编程语言验证的真实业务场景

## 场景1：SQL数据库题目

### 题目设置
```json
{
  "problem_id": 1001,
  "title": "查询员工薪资",
  "type": "database",
  "languages": ["sql"],           // 只允许SQL
  "description": "使用SQL查询薪资大于10000的员工"
}
```

### 系统支持的语言
```yaml
JudgeEngine:
  Compilers:
    cpp: {...}
    java: {...}
    python: {...}
    sql: {...}                   # 系统支持SQL执行
```

### 验证结果
- 用户提交Python代码 → ❌ **第3步失败**：题目不支持Python（业务限制）
- 用户提交SQL代码 → ✅ **第3步通过**，✅ **第4步通过**

---

## 场景2：算法竞赛题目

### 题目设置  
```json
{
  "problem_id": 1002,
  "title": "最短路径算法",
  "type": "algorithm", 
  "languages": ["cpp", "java"],   // 只允许C++和Java（性能要求）
  "time_limit": 1000,
  "description": "实现Dijkstra算法"
}
```

### 验证结果
- 用户提交Python代码 → ❌ **第3步失败**：题目不支持Python（性能限制）
- 用户提交C++代码 → ✅ **第3步通过**，✅ **第4步通过**

---

## 场景3：前端开发题目

### 题目设置
```json
{
  "problem_id": 1003,
  "title": "实现React组件",
  "type": "frontend",
  "languages": ["javascript", "typescript"], // 只允许前端语言
  "description": "创建一个可复用的Button组件"
}
```

### 验证结果
- 用户提交Java代码 → ❌ **第3步失败**：题目不支持Java（领域限制）
- 用户提交JavaScript代码 → ✅ **第3步通过**，✅ **第4步通过**

---

## 场景4：新语言支持问题

### 系统新增Rust支持
```yaml
JudgeEngine:
  Compilers:
    rust:                        # 新增Rust编译器
      CompileCommand: "rustc ..."
```

### 题目尚未更新
```json
{
  "problem_id": 1004,
  "title": "经典算法题",
  "languages": ["cpp", "java", "python"], // 题目尚未添加rust支持
}
```

### 验证结果
- 用户提交Rust代码 → ❌ **第3步失败**：题目不支持Rust（题目未更新）
- 系统支持Rust → ✅ **第4步会通过**

---

## 场景5：系统维护场景

### 临时关闭Python支持
```yaml
JudgeEngine:
  Compilers:
    # python: {...}              # Python编译器临时关闭维护
```

### 题目仍然配置支持Python
```json
{
  "problem_id": 1005,
  "languages": ["cpp", "java", "python"], // 题目配置支持python
}
```

### 验证结果
- 用户提交Python代码 → ✅ **第3步通过**，❌ **第4步失败**：系统不支持Python（维护中）

---

## 验证逻辑的必要性总结

### 第3步验证（题目业务限制）的价值：
1. **题目类型限制**：SQL题只能用SQL，前端题只能用JS/TS
2. **性能要求**：算法竞赛题限制高性能语言
3. **教学目标**：C语言课程题目只允许C语言
4. **安全考虑**：某些题目禁止使用特定语言的危险特性

### 第4步验证（系统技术限制）的价值：
1. **技术可行性**：系统是否具备执行该语言的能力
2. **维护状态**：某个编译器可能临时不可用
3. **版本兼容**：新语言支持可能还在测试阶段

## 优化建议

```go
// 优化后的验证逻辑，提供更明确的错误信息
func (l *SubmitJudgeLogic) validateLanguageSupport(language string, problemInfo *types.ProblemInfo) error {
    // 1. 先检查题目业务限制（更具体的错误信息）
    if !l.isLanguageSupportedByProblem(language, problemInfo.Languages) {
        return fmt.Errorf("该题目不支持 %s 语言，支持的语言：%v", 
            language, problemInfo.Languages)
    }
    
    // 2. 再检查系统技术限制
    supportedLanguages := l.svcCtx.JudgeEngine.GetSystemInfo()["supported_languages"].([]string)
    if !l.isLanguageSupported(language, supportedLanguages) {
        return fmt.Errorf("判题系统暂不支持 %s 语言（系统维护中），支持的语言：%v", 
            language, supportedLanguages)
    }
    
    return nil
}
```
