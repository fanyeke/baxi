# Marketing Funnel 数据集获取指南

## 数据集说明

Olist Marketing Funnel 数据集记录了 Olist 如何获取卖家（商家）以及这些卖家的转化过程，是理解 Olist 商业模式的关键数据。

## 包含文件

该数据集包含两个核心文件：

1. **olist_marketing_qualified_leads_dataset.csv** (营销合格线索 - MQL)
   - 记录通过营销渠道获取的潜在卖家
   - 包含约 8,000 条线索数据
   - 字段包括：线索来源、首次接触日期、landing_page 等

2. **olist_closed_deals_dataset.csv** (成交交易)
   - 记录成功转化为 Olist 平台卖家的记录
   - 包含卖家转化详情
   - 关联 seller_id，可连接到主数据集

## 数据集关系

```
营销漏斗流程:
MQL (营销合格线索)
  ↓ mql_id
Closed Deals (成交交易)
  ↓ seller_id
Sellers (卖家) [主数据集]
  ↓ seller_id
Order Items (订单商品) [主数据集]
```

## 下载步骤

### 方法 1: 从 Kaggle 下载

1. 访问 Kaggle 数据集页面：
   ```
   https://www.kaggle.com/datasets/olistbr/marketing-funnel
   ```

2. 点击 "Download" 下载压缩包

3. 解压后将以下文件放到项目根目录：
   - `olist_marketing_qualified_leads_dataset.csv`
   - `olist_closed_deals_dataset.csv`

### 方法 2: 使用 Kaggle API

如果已安装 Kaggle CLI:

```bash
# 下载 Marketing Funnel 数据集
kaggle datasets download -d olistbr/marketing-funnel

# 解压
unzip marketing-funnel.zip

# 移动到项目目录（如需要）
mv olist_marketing_qualified_leads_dataset.csv ./
mv olist_closed_deals_dataset.csv ./
```

## 验证数据

下载完成后，运行验证：

```bash
# 检查文件是否存在
ls olist_marketing_qualified_leads_dataset.csv
ls olist_closed_deals_dataset.csv

# 运行扩展分析脚本
python3 explore_data_extended.py
```

预期输出应包含：

```
营销漏斗数据集文件: 2/2
  ✓ olist_marketing_qualified_leads_dataset.csv
  ✓ olist_closed_deals_dataset.csv
```

## 分析产出

添加 Marketing Funnel 数据后，分析结果将更新为：

1. **outputs/data_profile_summary.csv**
   - 新增营销漏斗表的字段分析
   - 增加约 20+ 行数据（两个表的字段）

2. **docs/data_dictionary.md**
   - 新增"营销漏斗数据集"章节
   - 详细说明 MQL 和 Closed Deals 表结构

3. **docs/table_relationship_notes.md**
   - 新增"跨数据集关联"部分
   - 展示营销漏斗与主数据集的关联关系
   - 核心关联：seller_id 连接两个数据集

4. **README.md**
   - 更新数据集统计信息
   - 补充营销漏斗业务说明

## 业务价值

Marketing Funnel 数据集将帮助：

- 理解卖家获客渠道效果
- 分析线索转化率
- 评估不同 landing page 的表现
- 连接营销投入与销售业绩
- 建立完整的"营销 → 转化 → 销售"业务链路分析

## 注意事项

- 确保下载的文件名与上述完全一致
- 数据格式应为 CSV
- 如遇编码问题，脚本会自动处理 UTF-8 编码
- 不要修改原始数据文件

---

**更新日期**: 2026-05-20