#!/usr/bin/env python3
"""
扩展数据探索脚本 - 包含 Marketing Funnel 数据集分析
支持增量更新现有分析结果
"""
import pandas as pd
import os
from pathlib import Path
import json
from datetime import datetime
import warnings
warnings.filterwarnings('ignore')

# Marketing Funnel 数据集文件名
MARKETING_FUNNEL_FILES = [
    'olist_marketing_qualified_leads_dataset.csv',
    'olist_closed_deals_dataset.csv'
]

# Olist 主数据集文件名
MAIN_DATASET_FILES = [
    'olist_customers_dataset.csv',
    'olist_geolocation_dataset.csv',
    'olist_order_items_dataset.csv',
    'olist_order_payments_dataset.csv',
    'olist_order_reviews_dataset.csv',
    'olist_orders_dataset.csv',
    'olist_products_dataset.csv',
    'olist_sellers_dataset.csv',
    'product_category_name_translation.csv'
]

def infer_date_column(df, column):
    """尝试推断日期字段"""
    try:
        sample = df[column].dropna().head(100)
        if len(sample) > 0:
            parsed = pd.to_datetime(sample, errors='coerce')
            if parsed.notna().sum() / len(sample) > 0.8:
                full_parsed = pd.to_datetime(df[column], errors='coerce')
                return {
                    'is_date': True,
                    'min_date': str(full_parsed.min()),
                    'max_date': str(full_parsed.max())
                }
    except Exception:
        pass
    return {'is_date': False}

def profile_csv(filepath, dataset_type='main'):
    """对单个 CSV 文件进行画像分析"""
    print(f"正在分析: {filepath} [{dataset_type}]")

    df = pd.read_csv(filepath)
    profile = {
        'table_name': Path(filepath).stem,
        'filepath': filepath,
        'dataset_type': dataset_type,
        'row_count': len(df),
        'column_count': len(df.columns),
        'columns': [],
        'duplicate_rows': df.duplicated().sum(),
        'duplicate_percentage': f"{(df.duplicated().sum() / len(df) * 100):.2f}%",
        'possible_primary_keys': [],
        'possible_foreign_keys': []
    }

    # 分析每一列
    for col in df.columns:
        col_info = {
            'column_name': col,
            'dtype': str(df[col].dtype),
            'non_null_count': df[col].notna().sum(),
            'null_count': df[col].isna().sum(),
            'null_percentage': f"{(df[col].isna().sum() / len(df) * 100):.2f}%",
            'unique_count': df[col].nunique(),
            'unique_percentage': f"{(df[col].nunique() / len(df) * 100):.2f}%",
            'sample_values': df[col].dropna().head(3).tolist()
        }

        date_info = infer_date_column(df, col)
        if date_info['is_date']:
            col_info['is_date'] = True
            col_info['date_range'] = f"{date_info['min_date']} 到 {date_info['max_date']}"
        else:
            col_info['is_date'] = False

        # 判断主键
        if (df[col].notna().all() and
            df[col].nunique() == len(df) and
            len(df) > 0):
            col_info['possible_pk'] = True
            profile['possible_primary_keys'].append(col)
        else:
            col_info['possible_pk'] = False

        # 判断外键
        if (col.endswith('_id') or col.endswith('_key') or
            col.endswith('_code') or col.endswith('_zip')):
            col_info['possible_fk'] = True
            profile['possible_foreign_keys'].append(col)
        else:
            col_info['possible_fk'] = False

        profile['columns'].append(col_info)

    return profile

def analyze_relationships(profiles):
    """分析表之间的关系"""
    relationships = []

    for profile in profiles:
        table_name = profile['table_name']
        dataset_type = profile['dataset_type']

        for fk in profile['possible_foreign_keys']:
            for other_profile in profiles:
                if other_profile['table_name'] != table_name:
                    if fk in other_profile['possible_primary_keys']:
                        rel_type = '跨数据集关联' if dataset_type != other_profile['dataset_type'] else '主数据集关联'
                        relationships.append({
                            'from_table': table_name,
                            'from_dataset': dataset_type,
                            'from_column': fk,
                            'to_table': other_profile['table_name'],
                            'to_dataset': other_profile['dataset_type'],
                            'to_column': fk,
                            'relationship': f"{table_name}.{fk} -> {other_profile['table_name']}.{fk}",
                            'relationship_type': rel_type
                        })
                    elif fk in [col['column_name'] for col in other_profile['columns']]:
                        rel_type = '跨数据集关联(待验证)' if dataset_type != other_profile['dataset_type'] else '主数据集关联(待验证)'
                        relationships.append({
                            'from_table': table_name,
                            'from_dataset': dataset_type,
                            'from_column': fk,
                            'to_table': other_profile['table_name'],
                            'to_dataset': other_profile['dataset_type'],
                            'to_column': fk,
                            'relationship': f"{table_name}.{fk} -> {other_profile['table_name']}.{fk} (待验证)",
                            'relationship_type': rel_type
                        })

    return relationships

def generate_summary_csv(profiles, output_path):
    """生成汇总 CSV"""
    rows = []
    for profile in profiles:
        for col in profile['columns']:
            rows.append({
                '数据集类型': profile.get('dataset_type', 'main'),
                '表名': profile['table_name'],
                '列名': col['column_name'],
                '数据类型': col['dtype'],
                '总行数': profile['row_count'],
                '非空计数': col['non_null_count'],
                '缺失计数': col['null_count'],
                '缺失率': col['null_percentage'],
                '唯一值数': col['unique_count'],
                '唯一值率': col['unique_percentage'],
                '是否日期': col['is_date'],
                '日期范围': col.get('date_range', ''),
                '可能主键': col['possible_pk'],
                '可能外键': col['possible_fk'],
                '示例值': str(col['sample_values'][:3])
            })

    df = pd.DataFrame(rows)
    df.to_csv(output_path, index=False, encoding='utf-8')
    print(f"✓ 已生成汇总 CSV: {output_path} ({len(df)} 行)")

def generate_data_dictionary(profiles, output_path):
    """生成数据字典 Markdown"""
    content = "# Olist 数据字典\n\n"
    content += f"生成时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"
    content += "本数据字典包含两个数据集：\n"
    content += "- **主数据集** (Olist Brazilian E-Commerce): 9 张表\n"
    content += "- **营销漏斗数据集** (Marketing Funnel): 营销线索与成交数据\n\n"
    content += "---\n\n"

    # 按数据集类型分组
    main_profiles = [p for p in profiles if p.get('dataset_type') == 'main']
    marketing_profiles = [p for p in profiles if p.get('dataset_type') == 'marketing']

    # 主数据集
    if main_profiles:
        content += "## 主数据集 (Olist Brazilian E-Commerce)\n\n"
        for profile in main_profiles:
            content += generate_table_section(profile)
        content += "\n"

    # 营销漏斗数据集
    if marketing_profiles:
        content += "## 营销漏斗数据集 (Marketing Funnel)\n\n"
        content += "### 数据集概述\n\n"
        content += "营销漏斗数据集记录了 Olist 如何获取卖家（商家）以及这些卖家的转化过程。\n\n"
        content += "数据集包含两个核心表：\n"
        content += "- **营销合格线索 (MQL)**: 通过营销渠道获取的潜在卖家\n"
        content += "- **成交交易**: 成功转化为 Olist 平台卖家的记录\n\n"

        for profile in marketing_profiles:
            content += generate_table_section(profile)

    with open(output_path, 'w', encoding='utf-8') as f:
        f.write(content)
    print(f"✓ 已生成数据字典: {output_path}")

def generate_table_section(profile):
    """生成单个表的说明部分"""
    content = f"### {profile['table_name']}\n\n"
    content += f"- **文件路径**: `{profile['filepath']}`\n"
    content += f"- **总行数**: {profile['row_count']:,}\n"
    content += f"- **总列数**: {profile['column_count']}\n"
    content += f"- **重复行数**: {profile['duplicate_rows']} ({profile['duplicate_percentage']})\n"

    if profile['possible_primary_keys']:
        content += f"- **可能主键**: {', '.join(profile['possible_primary_keys'])}\n"

    if profile['possible_foreign_keys']:
        content += f"- **可能外键**: {', '.join(profile['possible_foreign_keys'])}\n"

    content += "\n#### 字段详情\n\n"
    content += "| 字段名 | 数据类型 | 非空计数 | 缺失率 | 唯一值数 | 唯一值率 | 是否日期 | 日期范围 | 可能主键 | 可能外键 |\n"
    content += "|--------|----------|----------|--------|----------|----------|----------|----------|----------|----------|\n"

    for col in profile['columns']:
        is_pk = "✓" if col['possible_pk'] else ""
        is_fk = "✓" if col['possible_fk'] else ""
        is_date = "✓" if col['is_date'] else ""
        date_range = col.get('date_range', '')

        content += f"| {col['column_name']} | {col['dtype']} | {col['non_null_count']:,} | {col['null_percentage']} | {col['unique_count']:,} | {col['unique_percentage']} | {is_date} | {date_range} | {is_pk} | {is_fk} |\n"

    content += "\n---\n\n"
    return content

def generate_relationship_notes(profiles, relationships, output_path):
    """生成表关系说明 Markdown"""
    content = "# Olist 表关系分析\n\n"
    content += f"生成时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"
    content += "---\n\n"

    # 表概览
    content += "## 表概览\n\n"
    content += "| 数据集 | 表名 | 行数 | 列数 | 可能主键 | 可能外键 |\n"
    content += "|--------|------|------|------|----------|----------|\n"

    for profile in profiles:
        dataset_type = profile.get('dataset_type', 'main')
        dataset_label = '主数据集' if dataset_type == 'main' else '营销漏斗'
        pk_str = ', '.join(profile['possible_primary_keys']) if profile['possible_primary_keys'] else '-'
        fk_str = ', '.join(profile['possible_foreign_keys']) if profile['possible_foreign_keys'] else '-'
        content += f"| {dataset_label} | {profile['table_name']} | {profile['row_count']:,} | {profile['column_count']} | {pk_str} | {fk_str} |\n"

    content += "\n---\n\n"

    # 关系分析
    content += "## 表关系推断\n\n"

    # 分类展示关系
    main_relations = [r for r in relationships if r['relationship_type'].startswith('主数据集')]
    cross_relations = [r for r in relationships if r['relationship_type'].startswith('跨数据集')]

    if main_relations:
        content += "### 主数据集内部关系\n\n"
        for rel in main_relations:
            content += f"- **{rel['relationship']}**\n"

    if cross_relations:
        content += "\n### 跨数据集关联（核心业务链路）\n\n"
        content += "营销漏斗数据集与主数据集的关联关系：\n\n"
        for rel in cross_relations:
            content += f"- **{rel['relationship']}**\n"
            content += f"  - 关联类型: {rel['relationship_type']}\n"
            content += f"  - 源表: `{rel['from_table']}` ({rel['from_dataset']})\n"
            content += f"  - 目标表: `{rel['to_table']}` ({rel['to_dataset']})\n\n"

    content += "\n---\n\n"

    # 数据质量
    content += "## 数据质量说明\n\n"

    main_profiles = [p for p in profiles if p.get('dataset_type') == 'main']
    marketing_profiles = [p for p in profiles if p.get('dataset_type') == 'marketing']

    if main_profiles:
        content += "### 主数据集\n\n"
        for profile in main_profiles:
            if profile['duplicate_rows'] > 0:
                content += f"- **{profile['table_name']}**: 存在 {profile['duplicate_rows']} 行重复数据 ({profile['duplicate_percentage']})\n"

            null_cols = [col for col in profile['columns'] if col['null_count'] > 0]
            if null_cols:
                content += f"- **{profile['table_name']}**: 以下字段存在缺失值\n"
                for col in null_cols:
                    content += f"  - `{col['column_name']}`: {col['null_count']} ({col['null_percentage']})\n"

    if marketing_profiles:
        content += "\n### 营销漏斗数据集\n\n"
        for profile in marketing_profiles:
            if profile['duplicate_rows'] > 0:
                content += f"- **{profile['table_name']}**: 存在 {profile['duplicate_rows']} 行重复数据 ({profile['duplicate_percentage']})\n"

            null_cols = [col for col in profile['columns'] if col['null_count'] > 0]
            if null_cols:
                content += f"- **{profile['table_name']}**: 以下字段存在缺失值\n"
                for col in null_cols:
                    content += f"  - `{col['column_name']}`: {col['null_count']} ({col['null_percentage']})\n"

    content += "\n---\n\n"
    content += "**注**: 以上关系为初步推断，需要通过实际查询验证。\n"

    with open(output_path, 'w', encoding='utf-8') as f:
        f.write(content)
    print(f"✓ 已生成表关系说明: {output_path}")

def scan_available_files():
    """扫描可用的 CSV 文件"""
    all_csv_files = [f for f in os.listdir('.') if f.endswith('.csv')]

    main_files = [f for f in all_csv_files if f in MAIN_DATASET_FILES]
    marketing_files = [f for f in all_csv_files if f in MARKETING_FUNNEL_FILES]

    return main_files, marketing_files

def main():
    """主函数"""
    print("=" * 80)
    print("Olist 数据集探索分析（含 Marketing Funnel）")
    print("=" * 80)
    print()

    # 扫描文件
    main_files, marketing_files = scan_available_files()

    print(f"主数据集文件: {len(main_files)}/{len(MAIN_DATASET_FILES)}")
    for f in main_files:
        print(f"  ✓ {f}")

    missing_main = [f for f in MAIN_DATASET_FILES if f not in main_files]
    if missing_main:
        print(f"  ✗ 缺失:")
        for f in missing_main:
            print(f"    - {f}")

    print(f"\n营销漏斗数据集文件: {len(marketing_files)}/{len(MARKETING_FUNNEL_FILES)}")
    for f in marketing_files:
        print(f"  ✓ {f}")

    missing_marketing = [f for f in MARKETING_FUNNEL_FILES if f not in marketing_files]
    if missing_marketing:
        print(f"  ✗ 缺失:")
        for f in missing_marketing:
            print(f"    - {f}")
        print("\n提示: Marketing Funnel 数据集需要从 Kaggle 下载:")
        print("  https://www.kaggle.com/datasets/olistbr/marketing-funnel")

    print()

    # 分析文件
    profiles = []

    if main_files:
        print("=" * 80)
        print("分析主数据集...")
        print("=" * 80)
        for csv_file in sorted(main_files):
            profile = profile_csv(csv_file, dataset_type='main')
            profiles.append(profile)

    if marketing_files:
        print("\n" + "=" * 80)
        print("分析营销漏斗数据集...")
        print("=" * 80)
        for csv_file in sorted(marketing_files):
            profile = profile_csv(csv_file, dataset_type='marketing')
            profiles.append(profile)

    if not profiles:
        print("\n⚠ 没有找到任何数据文件！")
        return

    print("\n" + "=" * 80)
    print("分析表关系...")
    print("=" * 80)

    relationships = analyze_relationships(profiles)

    # 统计关系类型
    main_rels = len([r for r in relationships if r['relationship_type'].startswith('主数据集')])
    cross_rels = len([r for r in relationships if r['relationship_type'].startswith('跨数据集')])
    print(f"发现 {len(relationships)} 个关系:")
    print(f"  - 主数据集内部关系: {main_rels}")
    print(f"  - 跨数据集关联: {cross_rels}")

    print("\n" + "=" * 80)
    print("生成报告...")
    print("=" * 80)

    generate_summary_csv(profiles, 'outputs/data_profile_summary.csv')
    generate_data_dictionary(profiles, 'docs/data_dictionary.md')
    generate_relationship_notes(profiles, relationships, 'docs/table_relationship_notes.md')

    print("\n" + "=" * 80)
    print("✓ 所有报告已生成完成！")
    print("=" * 80)
    print("\n输出文件:")
    print("  1. outputs/data_profile_summary.csv - 数据画像汇总")
    print("  2. docs/data_dictionary.md - 数据字典")
    print("  3. docs/table_relationship_notes.md - 表关系说明")

    if marketing_files:
        print("\n数据集统计:")
        print(f"  - 主数据集: {len(main_files)} 张表")
        print(f"  - 营销漏斗数据集: {len(marketing_files)} 张表")
        print(f"  - 总计: {len(profiles)} 张表")

if __name__ == '__main__':
    main()