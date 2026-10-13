# AGENTS.md

do-template 项目 Go 编码规范。所有代码必须遵循以下规则；规范之间冲突时，按 **规则 2（版本策略）> 规则 3（Uber 风格）> 规则 4（最佳实践）** 裁决，其余情况以 Go 官方文档为准。

## 1. 哲学与代码拆分

遵循 Unix 哲学、Go 惯用法与 UTF-8 编码规范：

- **Unix 哲学**：一个模块只做一件事，并把它做好；通过组合小模块解决复杂问题
- **UTF-8**：所有源码、文档、字符串一律 UTF-8 编码；文本处理按 rune 而非 byte
- **代码拆分原则**：
  - 按职责拆分：单一职责，高内聚、低耦合
  - 按包拆分：每个包职责清晰、可独立测试；避免包之间循环依赖
  - 按文件拆分：单文件过长时按功能拆出独立文件；导出符号配文档注释
  - 拆分粒度以复用和可读为准，不做投机性抽象

## 2. Go 版本策略

Go 一律使用最新版本的基础库、语法和特性：

- 优先使用标准库最新 API，禁止自行造轮子（如 `slices`、`maps`、`log/slog`、`errors.Join`、泛型等）
- 采用最新语言特性：`range over int`、`range over func`（迭代器）、泛型、`min/max/clear` 内置函数等
- 禁用已废弃的写法（如 `interface{}` 应写作 `any`，`ioutil` 已废弃改用 `os`/`io`）
- 新代码不得引入旧语法或为兼容旧版本而降级写法

## 3. 代码风格（Uber）

完全遵循 [Uber Go 风格指南](https://github.com/uber-go/guide)，与个人偏好冲突时一律以该指南为准。关键约束（非穷举，完整内容以指南为准）：

- 错误：包装用 `%w`；断言用 `errors.As`，比较用 `errors.Is`；避免在结构体上定义 `Error` 方法
- 包导入按标准库 / 第三方分组，禁用相对导入
- 函数按「接收上下文 → 参数 → 返回值」排列；`context.Context` 永远是第一个参数
- 避免函数式选项模式（functional options）滥用；构造函数返回 `T` 而非 `*T` 仅限小值类型
- 零值有用的类型优先（如 `bytes.Buffer`、`sync.Mutex`）；接口越小越好
- 测试使用表驱动（table-driven tests）；包名不用 `util`/`common` 这类无意义命名
- 命名用 MixedCaps 驼峰；缩写词全大写或全小写（`URL`/`url`，禁用 `Url`）
- 提交前可用 `golangci-lint`（启用指南相关规则集）自查

## 4. 最佳实践

以下三份资料同样必须遵循，与规则 3 重叠的条目以规则 3 为准。

### Go Proverbs（Rob Pike）— 设计哲学

- Clear is better than clever（清晰优于聪明）
- Simple is better than complex；少即是多
- Don't communicate by sharing memory, share memory by communicating（用 channel 通信）
- Don't just check errors, handle them gracefully
- A little copying is better than a little dependency

### Practical Go（Dave Cheney）— 错误处理与 API 设计

- 错误只处理一次：要么记日志并返回，要么向上包装返回，不要两者都做
- 错误信息小写开头、不带标点、不写 "failed to" 类冗余前缀；包装时补足上下文
- API 设计：避免 `func init()`、参数尽量少、延迟初始化
- 避免返回未初始化的 nil interface

### 100 Go Mistakes（Teiva Harsanyi）— 反模式清单（code review 检查表）

- 被遗忘的 `defer` 资源清理、loop 变量捕获（Go 1.22 前的闭包陷阱）
- `time.After` 在循环中泄漏、不必要的嵌套与 else-return
- 在值类型上误用指针（或反之）、返回未导出类型的切片
- 忽略 `context` 取消传播、`err` 被覆盖导致丢失
- 并发写 map、`sync.Mutex` 值拷贝、goroutine 泄漏（无退出条件）

## 参考资料汇总

| 资料 | 用途 | 地址 |
|---|---|---|
| Uber Go Style Guide | 工程风格基准 | github.com/uber-go/guide |
| Go Proverbs | 设计哲学 | go-proverbs.github.io |
| Practical Go | 错误处理 / API 设计 | dave.cheney.net/practical-go |
| 100 Go Mistakes | 反模式检查表 | 100go.co |
