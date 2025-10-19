# 智能预处理指南

## 概述

新版本支持自动从文件名提取BPM，简化预处理流程。

## 文件命名规范

为了让系统自动识别BPM，请按以下格式命名音乐文件：

```
{歌曲名}-{BPM}.json
```

### 示例

| 文件名 | 提取的BPM |
|--------|----------|
| `青花瓷-葫芦丝-4min-108.json` | 108 |
| `茉莉花-92.json` | 92 |
| `康定情歌-唢呐-100.json` | 100 |
| `梁祝-120.json` | 120 |

**规则：**
- 文件名最后一个 `-` 后面的数字就是BPM
- 数字必须在 `.json` 扩展名之前
- 例如：`xxx-108.json` → BPM = 108

## 使用方法

### 方式1：单文件智能预处理

```bash
# 自动提取BPM，使用默认参数（乐器=sn，吐音=20ms）
./preprocess_auto.sh trsmusic/青花瓷-葫芦丝-4min-108.json

# 指定乐器类型
./preprocess_auto.sh trsmusic/青花瓷-葫芦丝-4min-108.json sn

# 指定乐器和吐音延迟
./preprocess_auto.sh trsmusic/青花瓷-葫芦丝-4min-108.json sn 20
```

**执行过程：**
```
════════════════════════════════════════════════════════════════
            智能预处理（自动BPM识别）
════════════════════════════════════════════════════════════════

输入文件: 青花瓷-葫芦丝-4min-108.json

配置信息:
  ✓ BPM: 108
  ✓ 乐器类型: sn (唢呐)
  ✓ 吐音延迟: 20 ms

将生成文件: exec/青花瓷-葫芦丝-4min-108_sn_108_20.exec.json

是否继续? (y/n)
```

### 方式2：批量智能预处理

```bash
# 使用默认参数（乐器=sn，吐音=20ms）
./batch_preprocess.sh

# 指定乐器类型
./batch_preprocess.sh sks

# 指定乐器和吐音延迟
./batch_preprocess.sh sn 20
```

**执行过程：**
```
════════════════════════════════════════════════════════════════
            批量预处理音乐文件（智能BPM识别）
════════════════════════════════════════════════════════════════

参数配置:
  乐器类型: sn (sn=唢呐, sks=萨克斯)
  吐音延迟: 20 ms
  BPM: 自动从文件名提取

找到 5 个音乐文件

→ 处理: 青花瓷-葫芦丝-4min-108.json
  📊 提取BPM: 108
  ✓ 成功: exec/青花瓷-葫芦丝-4min-108_sn_108_20.exec.json

→ 处理: 茉莉花-92.json
  📊 提取BPM: 92
  ✓ 成功: exec/茉莉花_sn_92_20.exec.json
```

### 方式3：传统方式（手动指定BPM）

如果需要覆盖文件名中的BPM，仍可以手动指定：

```bash
./newsksgo -preprocess \
  -in trsmusic/青花瓷-葫芦丝-4min-108.json \
  -instrument sn \
  -bpm 120 \
  -tongue 20
```

## 默认参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| **乐器类型** | `sn` | sn=唢呐, sks=萨克斯 |
| **吐音延迟** | `20` | 单位：毫秒 |
| **BPM** | 自动提取 | 从文件名自动识别 |

## 完整工作流

### 步骤1：准备音乐文件

确保文件名包含BPM：

```bash
# 查看音乐文件
ls trsmusic/

青花瓷-葫芦丝-4min-108.json
茉莉花-92.json
康定情歌-唢呐-100.json
```

### 步骤2：批量预处理

```bash
# 生成所有唢呐版本（吐音延迟20ms）
./batch_preprocess.sh sn 20

# 生成所有萨克斯版本（吐音延迟25ms）
./batch_preprocess.sh sks 25
```

### 步骤3：播放

```bash
# 播放生成的文件
./newsksgo -json exec/青花瓷-葫芦丝-4min-108_sn_108_20.exec.json
```

## 文件名格式说明

生成的exec文件仍然遵循标准命名规则：

```
exec/{原文件名}_{乐器类型}_{BPM}_{吐音延迟}.exec.json
```

### 示例

| 输入 | 参数 | 输出 |
|------|------|------|
| `青花瓷-葫芦丝-4min-108.json` | `sn, 20` | `exec/青花瓷-葫芦丝-4min-108_sn_108_20.exec.json` |
| `茉莉花-92.json` | `sks, 25` | `exec/茉莉花_sks_92_25.exec.json` |

## 常见问题

### Q1: 文件名不包含BPM怎么办？

**方式1：重命名文件**
```bash
mv trsmusic/茉莉花.json trsmusic/茉莉花-92.json
```

**方式2：使用传统方式手动指定**
```bash
./newsksgo -preprocess -in trsmusic/茉莉花.json -instrument sn -bpm 92 -tongue 20
```

**方式3：在脚本运行时输入**
```bash
./preprocess_auto.sh trsmusic/茉莉花.json
# 脚本会提示你输入BPM
```

### Q2: 如何为同一首歌生成多个BPM版本？

手动指定不同的BPM：

```bash
# 生成BPM=100的版本
./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm 100 -tongue 20

# 生成BPM=108的版本（原始BPM）
./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm 108 -tongue 20

# 生成BPM=120的版本
./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm 120 -tongue 20
```

### Q3: 批量预处理时，不同文件有不同BPM怎么办？

这正是智能预处理的优势！每个文件会自动使用自己的BPM：

```bash
./batch_preprocess.sh sn 20

# 输出：
# → 青花瓷-葫芦丝-4min-108.json  → BPM=108
# → 茉莉花-92.json              → BPM=92
# → 康定情歌-唢呐-100.json      → BPM=100
```

## 快速参考

### 单文件预处理

```bash
# 基本用法
./preprocess_auto.sh trsmusic/青花瓷-葫芦丝-4min-108.json

# 完整参数
./preprocess_auto.sh trsmusic/青花瓷-葫芦丝-4min-108.json sn 20
```

### 批量预处理

```bash
# 唢呐版本
./batch_preprocess.sh sn 20

# 萨克斯版本
./batch_preprocess.sh sks 25
```

### 播放

```bash
./newsksgo -json exec/青花瓷-葫芦丝-4min-108_sn_108_20.exec.json
```

### 查看生成的文件

```bash
# 查看所有exec文件
ls -lh exec/

# 查看特定配置的文件
ls exec/*_sn_*_20.exec.json
```

## 优势

1. **自动化** - 无需手动指定BPM
2. **批量处理** - 一次处理所有文件
3. **统一规范** - 文件命名标准化
4. **避免错误** - 减少人工输入错误
5. **易于管理** - 从文件名即可识别参数

## 注意事项

1. 文件名必须以 `-数字.json` 结尾才能自动识别BPM
2. 如果识别失败，脚本会提示手动输入或使用默认值
3. 建议统一使用标准命名格式，方便管理
4. 吐音延迟默认为20ms，这是经过测试的最佳值

