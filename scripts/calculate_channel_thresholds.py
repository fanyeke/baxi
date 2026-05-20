#!/usr/bin/env python3
"""
计算渠道分类的正确阈值

基于中位数分类规则：
- High Conversion: 转化率 >= 中位数
- High Value: GMV/商家 >= 中位数
"""

import pandas as pd
import numpy as np

# 读取数据
print("读取数据...")
mql_df = pd.read_csv('data/olist_marketing_qualified_leads_dataset.csv')
closed_df = pd.read_csv('data/olist_closed_deals_dataset.csv')
item_df = pd.read_csv('data/interim/item_level_base.csv')

print(f"MQL总数: {len(mql_df)}")
print(f"Closed Deals总数: {len(closed_df)}")
print(f"Item总数: {len(item_df)}")

# 1. 计算各来源的MQL数量
print("\n=== 1. MQL按来源分布 ===")
mql_by_origin = mql_df.groupby('origin').size().reset_index(name='mql_count')
mql_by_origin = mql_by_origin.sort_values('mql_count', ascending=False)
print(mql_by_origin)

# 2. 计算各来源的转化数量（Closed Deals）
print("\n=== 2. Closed Deals按来源分布 ===")
# 合并MQL和Closed Deals以获取来源信息
mql_with_origin = mql_df[['mql_id', 'origin']]
closed_with_origin = closed_df[['mql_id', 'seller_id']].merge(
    mql_with_origin, on='mql_id', how='left'
)

# 统计各来源的转化数量
conversion_by_origin = closed_with_origin.groupby('origin').size().reset_index(name='conversion_count')
print(conversion_by_origin)

# 3. 计算转化率
print("\n=== 3. 各来源转化率 ===")
conversion_rate_df = mql_by_origin.merge(conversion_by_origin, on='origin', how='left')
conversion_rate_df['conversion_count'] = conversion_rate_df['conversion_count'].fillna(0)
conversion_rate_df['conversion_rate'] = conversion_rate_df['conversion_count'] / conversion_rate_df['mql_count']
conversion_rate_df = conversion_rate_df.sort_values('conversion_rate', ascending=False)
print(conversion_rate_df)

# 计算转化率中位数
conversion_rate_median = conversion_rate_df['conversion_rate'].median()
print(f"\n转化率中位数: {conversion_rate_median:.4f} ({conversion_rate_median*100:.2f}%)")

# 4. 计算GMV/商家
print("\n=== 4. 计算GMV/商家 ===")
# 先计算每个商家的GMV
item_df['gmv'] = item_df['price'] + item_df['freight_value']
seller_gmv = item_df.groupby('seller_id')['gmv'].sum().reset_index()
seller_gmv.columns = ['seller_id', 'total_gmv']
print(f"有GMV记录的商家数: {len(seller_gmv)}")

# 合并商家GMV和来源信息
seller_with_origin = closed_with_origin.merge(seller_gmv, on='seller_id', how='left')
seller_with_origin['total_gmv'] = seller_with_origin['total_gmv'].fillna(0)

# 按来源计算平均GMV/商家
gmv_per_seller_by_origin = seller_with_origin.groupby('origin').agg({
    'seller_id': 'count',
    'total_gmv': 'sum'
}).reset_index()
gmv_per_seller_by_origin.columns = ['origin', 'seller_count', 'total_gmv']
gmv_per_seller_by_origin['gmv_per_seller'] = gmv_per_seller_by_origin['total_gmv'] / gmv_per_seller_by_origin['seller_count']
gmv_per_seller_by_origin = gmv_per_seller_by_origin.sort_values('gmv_per_seller', ascending=False)
print(gmv_per_seller_by_origin)

# 计算GMV/商家中位数
gmv_per_seller_median = gmv_per_seller_by_origin['gmv_per_seller'].median()
print(f"\nGMV/商家中位数: {gmv_per_seller_median:.2f}")

# 5. 基于中位数分类渠道
print("\n=== 5. 渠道分类结果 ===")
# 合并所有指标
channel_stats = conversion_rate_df.merge(gmv_per_seller_by_origin, on='origin', how='left')
channel_stats['gmv_per_seller'] = channel_stats['gmv_per_seller'].fillna(0)

# 分类
channel_stats['is_high_conversion'] = channel_stats['conversion_rate'] >= conversion_rate_median
channel_stats['is_high_value'] = channel_stats['gmv_per_seller'] >= gmv_per_seller_median

def classify_channel(row):
    if row['is_high_conversion'] and row['is_high_value']:
        return 'High Conversion & High Value'
    elif row['is_high_conversion']:
        return 'High Conversion & Low Value'
    elif row['is_high_value']:
        return 'Low Conversion & High Value'
    else:
        return 'Low Conversion & Low Value'

channel_stats['category'] = channel_stats.apply(classify_channel, axis=1)

# 显示结果
result = channel_stats[['origin', 'mql_count', 'conversion_count', 'conversion_rate', 
                         'gmv_per_seller', 'is_high_conversion', 'is_high_value', 'category']]
result = result.sort_values(['category', 'conversion_rate'], ascending=[True, False])
print(result.to_string())

# 6. 分类汇总
print("\n=== 6. 分类汇总 ===")
summary = channel_stats.groupby('category').agg({
    'origin': 'count',
    'mql_count': 'sum',
    'conversion_count': 'sum',
    'total_gmv': 'sum'
}).reset_index()
summary.columns = ['Category', '渠道数量', 'MQL总数', '转化总数', '总GMV']
print(summary)

# 7. 保存详细结果
print("\n=== 7. 保存结果 ===")
output_file = 'data/interim/channel_classification_corrected.csv'
result.to_csv(output_file, index=False)
print(f"结果已保存到: {output_file}")

# 8. 输出关键指标
print("\n" + "="*80)
print("关键指标汇总")
print("="*80)
print(f"转化率中位数阈值: {conversion_rate_median:.4f} ({conversion_rate_median*100:.2f}%)")
print(f"GMV/商家中位数阈值: {gmv_per_seller_median:.2f}")
print("\n各渠道分类:")
for _, row in result.iterrows():
    print(f"  {row['origin']:20s}: {row['category']}")