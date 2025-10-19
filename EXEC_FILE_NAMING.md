# Exec 文件命名规则

## 自动命名格式

执行序列文件（exec.json）使用以下命名格式：

```
exec/{原文件名}_{乐器类型}_{BPM}_{吐音延迟}.exec.json
```

### 示例

| 输入文件 | 参数 | 输出文件 |
|---------|------|---------|
| `trsmusic/青花瓷-葫芦丝-4min-108.json` | `-instrument sn -bpm 108 -tongue 30` | `exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json` |
| `trsmusic/茉莉花.json` | `-instrument sks -bpm 120 -tongue 25` | `exec/茉莉花_sks_120_25.exec.json` |
| `trsmusic/康定情歌-唢呐.json` | `-instrument sn -bpm 100 -tongue 30` | `exec/康定情歌-唢呐_sn_100_30.exec.json` |

## 文件名各部分说明

### 1. 原文件名
- 从输入文件路径提取，去除路径和 `.json` 扩展名
- 示例：`trsmusic/青花瓷-葫芦丝-4min-108.json` → `青花瓷-葫芦丝-4min-108`

### 2. 乐器类型
- `sn` = 唢呐 (Suona)
- `sks` = 萨克斯 (Saxophone)
- 由 `-instrument` 参数指定

### 3. BPM（每分钟节拍数）
- 取整数值（无小数）
- 由 `-bpm` 参数指定
- 如果未指定，使用配置文件中的默认值
- 示例：`108`, `120`, `92`

### 4. 吐音延迟
- 单位：毫秒（ms）
- 由 `-tongue` 参数指定
- 默认值：30
- 常用值：20-50 毫秒

## 使用方法

### 方式1：自动命名（推荐）

只需指定输入文件和参数，系统自动生成文件名并存放在 `exec/` 目录：

```bash
./newsksgo -preprocess \
  -in trsmusic/青花瓷-葫芦丝-4min-108.json \
  -instrument sn \
  -bpm 108 \
  -tongue 30
```

**输出：**
```
📝 自动生成输出文件名: exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json
✅ 预处理完成！
```

### 方式2：手动指定文件名

使用 `-out` 参数指定自定义输出路径：

```bash
./newsksgo -preprocess \
  -in trsmusic/茉莉花.json \
  -instrument sks \
  -bpm 120 \
  -out exec/my_custom_song.exec.json
```

## 为什么采用这种命名规则？

### 优点

1. **参数可追溯** - 从文件名即可知道生成参数，无需查看文件内容
2. **避免冲突** - 同一首歌的不同配置不会覆盖
3. **易于管理** - 批量查找特定配置的文件
4. **便于调试** - 快速识别测试不同参数的结果

### 应用场景

**场景1：测试不同BPM**
```bash
# 生成多个不同BPM的版本
./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm 100 -tongue 30
# → exec/青花瓷-葫芦丝-4min-108_sn_100_30.exec.json

./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm 108 -tongue 30
# → exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json

./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm 120 -tongue 30
# → exec/青花瓷-葫芦丝-4min-108_sn_120_30.exec.json
```

**场景2：测试不同吐音延迟**
```bash
./newsksgo -preprocess -in trsmusic/茉莉花.json -instrument sks -bpm 120 -tongue 20
# → exec/茉莉花_sks_120_20.exec.json

./newsksgo -preprocess -in trsmusic/茉莉花.json -instrument sks -bpm 120 -tongue 30
# → exec/茉莉花_sks_120_30.exec.json

./newsksgo -preprocess -in trsmusic/茉莉花.json -instrument sks -bpm 120 -tongue 40
# → exec/茉莉花_sks_120_40.exec.json
```

**场景3：同一首歌不同乐器**
```bash
# 萨克斯版本
./newsksgo -preprocess -in trsmusic/康定情歌.json -instrument sks -bpm 100 -tongue 30
# → exec/康定情歌_sks_100_30.exec.json

# 唢呐版本
./newsksgo -preprocess -in trsmusic/康定情歌.json -instrument sn -bpm 100 -tongue 30
# → exec/康定情歌_sn_100_30.exec.json
```

## 文件管理技巧

### 查找特定配置的文件

```bash
# 查找所有唢呐的文件
ls exec/*_sn_*.exec.json

# 查找所有BPM=108的文件
ls exec/*_*_108_*.exec.json

# 查找所有吐音延迟=30的文件
ls exec/*_*_*_30.exec.json

# 查找特定歌曲的所有版本
ls exec/青花瓷-*
```

### 批量删除

```bash
# 删除所有萨克斯版本
rm exec/*_sks_*.exec.json

# 删除特定歌曲的所有版本
rm exec/青花瓷-*

# 清空所有exec文件
rm exec/*.exec.json
```

### 批量生成

```bash
# 为所有歌曲生成唢呐版本
for file in trsmusic/*.json; do
  ./newsksgo -preprocess -in "$file" -instrument sn -bpm 108 -tongue 30
done

# 为一首歌生成多个BPM版本
for bpm in 100 108 120; do
  ./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm $bpm -tongue 30
done
```

## 注意事项

1. **文件名长度** - 如果原文件名很长，生成的文件名可能超出某些文件系统限制
2. **特殊字符** - 文件名中包含的特殊字符（如空格、中文）在某些环境可能需要转义
3. **覆盖警告** - 相同参数会生成相同文件名，会覆盖已存在的文件

## 相关命令

```bash
# 查看帮助
./newsksgo -help

# 查看exec目录内容
ls -lh exec/

# 查看文件详情
file exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json

# 查看文件大小
du -h exec/*.exec.json
```

