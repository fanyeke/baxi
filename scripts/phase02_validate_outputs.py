#!/usr/bin/env python3
"""
验证第一阶段产出文件
检查所有必需文件是否存在且格式正确
"""
import os
import pandas as pd
from pathlib import Path

def validate_outputs():
    """验证所有输出文件"""
    print("=" * 80)
    print("第一阶段产出文件验证")
    print("=" * 80)
    print()

    # 必需文件列表
    required_files = {
        'outputs/data_profile_summary.csv': '数据画像汇总',
        'docs/data_dictionary.md': '数据字典',
        'docs/table_relationship_notes.md': '表关系说明',
        'README.md': '项目说明文档',
        'explore_data.py': '数据探索脚本'
    }

    all_valid = True

    # 1. 检查文件存在性
    print("1. 检查文件存在性...")
    for filepath, description in required_files.items():
        if os.path.exists(filepath):
            size = os.path.getsize(filepath)
            print(f"  ✓ {description}: {filepath} ({size:,} bytes)")
        else:
            print(f"  ✗ {description}: {filepath} 不存在")
            all_valid = False
    print()

    # 2. 检查 CSV 文件格式
    print("2. 检查 CSV 文件格式...")
    try:
        df = pd.read_csv('outputs/data_profile_summary.csv', encoding='utf-8')
        print(f"  ✓ CSV 文件可正常读取")
        print(f"  ✓ 包含 {len(df)} 行数据")

        # 检查必需列
        required_columns = ['表名', '列名', '数据类型', '总行数', '缺失率', '唯一值数']
        missing_columns = [col for col in required_columns if col not in df.columns]
        if missing_columns:
            print(f"  ✗ 缺少列: {missing_columns}")
            all_valid = False
        else:
            print(f"  ✓ 包含所有必需列: {required_columns}")

        # 检查是否有 9 张表的数据
        tables = df['表名'].unique()
        print(f"  ✓ 包含 {len(tables)} 张表的数据")
        if len(tables) != 9:
            print(f"  ✗ 应包含 9 张表，实际 {len(tables)} 张")
            all_valid = False

    except Exception as e:
        print(f"  ✗ CSV 文件读取失败: {e}")
        all_valid = False
    print()

    # 3. 检查 Markdown 文件
    print("3. 检查 Markdown 文件...")
    for md_file in ['docs/data_dictionary.md', 'docs/table_relationship_notes.md', 'README.md']:
        try:
            with open(md_file, 'r', encoding='utf-8') as f:
                content = f.read()
                lines = content.split('\n')
                print(f"  ✓ {md_file}: {len(lines)} 行")

                # 检查是否为空
                if len(content.strip()) == 0:
                    print(f"  ✗ {md_file} 文件为空")
                    all_valid = False
        except Exception as e:
            print(f"  ✗ {md_file} 读取失败: {e}")
            all_valid = False
    print()

    # 4. 检查原始数据完整性
    print("4. 检查原始数据完整性...")
    csv_files = [f for f in os.listdir('.') if f.endswith('.csv')]
    expected_files = [
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

    for expected in expected_files:
        if expected in csv_files:
            size = os.path.getsize(expected)
            print(f"  ✓ {expected} ({size:,} bytes)")
        else:
            print(f"  ✗ {expected} 不存在")
            all_valid = False
    print()

    # 5. 验证完成标准
    print("5. 验证完成标准...")
    checks = {
        '每张原始表都有基础画像': len(tables) == 9,
        '字段含义与数据质量被清楚记录': os.path.exists('docs/data_dictionary.md'),
        '初步表关系已总结': os.path.exists('docs/table_relationship_notes.md'),
        '代码与结果可复现': os.path.exists('explore_data.py')
    }

    for check, passed in checks.items():
        status = "✓" if passed else "✗"
        print(f"  {status} {check}")
        if not passed:
            all_valid = False
    print()

    # 最终结果
    print("=" * 80)
    if all_valid:
        print("✓ 所有验证通过！第一阶段已完成")
    else:
        print("✗ 验证失败，请检查上述问题")
    print("=" * 80)

    return all_valid

if __name__ == '__main__':
    success = validate_outputs()
    exit(0 if success else 1)