#!/usr/bin/env python3
"""
Olist 数据集探索脚本
自动扫描所有 CSV 文件并生成数据画像
"""
import pandas as pd
import os
from pathlib import Path
import json
from datetime import datetime
import warnings
warnings.filterwarnings('ignore')

def infer_date_column(df, column):
    """尝试推断日期字段"""
    try:
        sample = df[column].dropna().head(100)
        if len(sample) > 0:
            # 尝试解析为日期
            parsed = pd.to_datetime(sample, errors='coerce')
            if parsed.notna().sum() / len(sample) > 0.8:  # 80%以上能解析
                full_parsed = pd.to_datetime(df[column], errors='coerce')
                return {
                    'is_date': True,
                    'min_date': str(full_parsed.min()),
                    'max_date': str(full_parsed.max())
                }
    except:
        pass
    return {'is_date': False}

def profile_csv(filepath):
    """对单个 CSV 文件进行画像分析"""
    print(f"正在分析: {filepath}")

    df = pd.read_csv(filepath)
    profile = {
        'table_name': Path(filepath).stem,
        'filepath': filepath,
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

        # 检查是否为日期字段
        date_info = infer_date_column(df, col)
        if date_info['is_date']:
            col_info['is_date'] = True
            col_info['date_range'] = f"{date_info['min_date']} 到 {date_info['max_date']}"
        else:
            col_info['is_date'] = False

        # 判断是否可能是主键
        if (df[col].notna().all() and
            df[col].nunique() == len(df) and
            len(df) > 0):
            col_info['possible_pk'] = True
            profile['possible_primary_keys'].append(col)
        else:
            col_info['possible_pk'] = False

        # 判断是否可能是外键（通常以 _id 或 _key 结尾）
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

    # 外键关系分析
    for profile in profiles:
        table_name = profile['table_name']
        for fk in profile['possible_foreign_keys']:
            # 查找其他表中是否有对应的主键
            for other_profile in profiles:
                if other_profile['table_name'] != table_name:
                    if fk in other_profile['possible_primary_keys']:
                        relationships.append({
                            'from_table': table_name,
                            'from_column': fk,
                            'to_table': other_profile['table_name'],
                            'to_column': fk,
                            'relationship': f"{table_name}.{fk} -> {other_profile['table_name']}.{fk}"
                        })
                    elif fk in [col['column_name'] for col in other_profile['columns']]:
                        relationships.append({
                            'from_table': table_name,
                            'from_column': fk,
                            'to_table': other_profile['table_name'],
                            'to_column': fk,
                            'relationship': f"{table_name}.{fk} -> {other_profile['table_name']}.{fk} (待验证)"
                        })

    return relationships

def generate_summary_csv(profiles, output_path):
    """生成汇总 CSV"""
    rows = []
    for profile in profiles:
        for col in profile['columns']:
            rows.append({
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
    print(f"✓ 已生成汇总 CSV: {output_path}")

def generate_data_dictionary(profiles, output_path):
    """生成数据字典 Markdown"""
    content = "# Olist 数据字典\n\n"
    content += f"生成时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"
    content += "---\n\n"

    for profile in profiles:
        content += f"## {profile['table_name']}\n\n"
        content += f"- **文件路径**: `{profile['filepath']}`\n"
        content += f"- **总行数**: {profile['row_count']:,}\n"
        content += f"- **总列数**: {profile['column_count']}\n"
        content += f"- **重复行数**: {profile['duplicate_rows']} ({profile['duplicate_percentage']})\n"

        if profile['possible_primary_keys']:
            content += f"- **可能主键**: {', '.join(profile['possible_primary_keys'])}\n"

        if profile['possible_foreign_keys']:
            content += f"- **可能外键**: {', '.join(profile['possible_foreign_keys'])}\n"

        content += "\n### 字段详情\n\n"
        content += "| 字段名 | 数据类型 | 非空计数 | 缺失率 | 唯一值数 | 唯一值率 | 是否日期 | 日期范围 | 可能主键 | 可能外键 |\n"
        content += "|--------|----------|----------|--------|----------|----------|----------|----------|----------|----------|\n"

        for col in profile['columns']:
            is_pk = "✓" if col['possible_pk'] else ""
            is_fk = "✓" if col['possible_fk'] else ""
            is_date = "✓" if col['is_date'] else ""
            date_range = col.get('date_range', '')

            content += f"| {col['column_name']} | {col['dtype']} | {col['non_null_count']:,} | {col['null_percentage']} | {col['unique_count']:,} | {col['unique_percentage']} | {is_date} | {date_range} | {is_pk} | {is_fk} |\n"

        content += "\n---\n\n"

    with open(output_path, 'w', encoding='utf-8') as f:
        f.write(content)
    print(f"✓ 已生成数据字典: {output_path}")

def generate_relationship_notes(profiles, relationships, output_path):
    """生成表关系说明 Markdown"""
    content = "# Olist 表关系分析\n\n"
    content += f"生成时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"
    content += "---\n\n"

    # 表概览
    content += "## 表概览\n\n"
    content += "| 表名 | 行数 | 列数 | 可能主键 | 可能外键 |\n"
    content += "|------|------|------|----------|----------|\n"

    for profile in profiles:
        pk_str = ', '.join(profile['possible_primary_keys']) if profile['possible_primary_keys'] else '-'
        fk_str = ', '.join(profile['possible_foreign_keys']) if profile['possible_foreign_keys'] else '-'
        content += f"| {profile['table_name']} | {profile['row_count']:,} | {profile['column_count']} | {pk_str} | {fk_str} |\n"

    content += "\n---\n\n"

    # 关系分析
    content += "## 表关系推断\n\n"
    content += "基于字段名称和数据分析，推断以下表关系：\n\n"

    if relationships:
        content += "### 外键关系\n\n"
        for rel in relationships:
            content += f"- **{rel['relationship']}**\n"
            content += f"  - 源表: `{rel['from_table']}`\n"
            content += f"  - 源字段: `{rel['from_column']}`\n"
            content += f"  - 目标表: `{rel['to_table']}`\n"
            content += f"  - 目标字段: `{rel['to_column']}`\n\n"
    else:
        content += "未发现明确的外键关系。\n"

    content += "\n---\n\n"

    # 数据质量说明
    content += "## 数据质量说明\n\n"
    for profile in profiles:
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

def main():
    """主函数"""
    print("=" * 80)
    print("Olist 数据集探索分析")
    print("=" * 80)
    print()

    # 查找所有 CSV 文件
    csv_files = sorted([f for f in os.listdir('.') if f.endswith('.csv')])
    print(f"发现 {len(csv_files)} 个 CSV 文件:")
    for f in csv_files:
        print(f"  - {f}")
    print()

    # 分析每个文件
    profiles = []
    for csv_file in csv_files:
        profile = profile_csv(csv_file)
        profiles.append(profile)

    print("\n" + "=" * 80)
    print("分析表关系...")
    print("=" * 80)

    # 分析关系
    relationships = analyze_relationships(profiles)

    print("\n" + "=" * 80)
    print("生成报告...")
    print("=" * 80)

    # 生成输出文件
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

if __name__ == '__main__':
    main()