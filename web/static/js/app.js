// 全局变量
let selectedFile = null;
let isPlaying = false;
let isPaused = false;
let autoScroll = true;
let statusUpdateInterval = null;
let logUpdateInterval = null;
let currentInstrument = 'sks'; // 当前选择的乐器：sks(萨克斯) 或 sn(唢呐)
let currentTimeline = null; // 当前加载的时间轴数据
let editingRestIndex = -1; // 正在编辑的空拍索引

// DOM元素
const searchInput = document.getElementById('searchInput');
const searchBtn = document.getElementById('searchBtn');
const fileList = document.getElementById('fileList');
const startBtn = document.getElementById('startBtn');
const pauseBtn = document.getElementById('pauseBtn');
const stopBtn = document.getElementById('stopBtn');
const clearLogBtn = document.getElementById('clearLogBtn');
const autoScrollBtn = document.getElementById('autoScrollBtn');
const logContent = document.getElementById('logContent');
const loadFingeringsBtn = document.getElementById('loadFingeringsBtn');
const fingeringButtonsEl = document.getElementById('fingeringButtons');

// 乐器切换元素
const sksBtn = document.getElementById('sksBtn');
const snBtn = document.getElementById('snBtn');

// 状态显示元素
const currentFileEl = document.getElementById('currentFile');
const progressEl = document.getElementById('progress');
const currentNoteEl = document.getElementById('currentNote');
const totalNotesEl = document.getElementById('totalNotes');
const elapsedTimeEl = document.getElementById('elapsedTime');
const playStatusEl = document.getElementById('playStatus');
const progressBarEl = document.getElementById('progressBar');

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    loadMusicFiles();
    setupEventListeners();
    startStatusUpdates();
    startLogUpdates();
    loadFingerings(); // 自动加载指法
    
    // 初始化模态框事件监听
    initModalListeners();
    
    // 初始化BPM输入监听
    initBpmListener();
});

// 设置事件监听器
function setupEventListeners() {
    // 搜索功能
    searchBtn.addEventListener('click', function() {
        loadMusicFiles(searchInput.value);
    });
    
    searchInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            loadMusicFiles(searchInput.value);
        }
    });
    
    // 控制按钮
    startBtn.addEventListener('click', startPlayback);
    pauseBtn.addEventListener('click', pausePlayback);
    stopBtn.addEventListener('click', stopPlayback);
    
    // 日志控制
    clearLogBtn.addEventListener('click', clearLogs);
    autoScrollBtn.addEventListener('click', toggleAutoScroll);
    
    // 指法测试按钮
    loadFingeringsBtn.addEventListener('click', loadFingerings);
    
    // 乐器切换按钮
    sksBtn.addEventListener('click', function() {
        switchInstrument('sks');
    });
    snBtn.addEventListener('click', function() {
        switchInstrument('sn');
    });
}

// 加载音乐文件列表
async function loadMusicFiles(search = '') {
    try {
        fileList.innerHTML = '<div class="loading">加载中...</div>';
        
        const response = await fetch(`/api/files?search=${encodeURIComponent(search)}`);
        const data = await response.json();
        
        if (data.error) {
            fileList.innerHTML = `<div class="error">错误: ${data.error}</div>`;
            return;
        }
        
        renderFileList(data.files);
    } catch (error) {
        console.error('加载文件列表失败:', error);
        fileList.innerHTML = '<div class="error">加载失败，请检查网络连接</div>';
    }
}

// 渲染文件列表
function renderFileList(files) {
    if (files.length === 0) {
        fileList.innerHTML = '<div class="no-files">没有找到音乐文件</div>';
        return;
    }
    
    fileList.innerHTML = '';
    
    files.forEach(file => {
        const fileItem = document.createElement('div');
        fileItem.className = 'file-item';
        fileItem.innerHTML = `
            <h4>${file.title}</h4>
            <div class="file-info">
                <span>📁 ${file.filename}</span>
                <span>🎵 ${file.bpm} BPM</span>
                <span>🎼 ${file.duration} 音符</span>
                <span>📅 ${file.modified_at}</span>
            </div>
        `;
        
        fileItem.addEventListener('click', function() {
            selectFile(file);
        });
        
        fileList.appendChild(fileItem);
    });
}

// 选择文件
function selectFile(file) {
    // 移除之前的选中状态
    document.querySelectorAll('.file-item').forEach(item => {
        item.classList.remove('selected');
    });
    
    // 添加选中状态
    event.currentTarget.classList.add('selected');
    selectedFile = file;
    
    // 更新开始按钮状态
    updateStartButtonState();
    
    // 加载歌曲时间轴
    loadSongTimeline(file.filename);
}

// 更新开始按钮状态
function updateStartButtonState() {
    startBtn.disabled = !selectedFile || isPlaying;
}

// 开始演奏
async function startPlayback() {
    if (!selectedFile || isPlaying) return;
    
    try {
        startBtn.disabled = true;
        
        // 获取用户输入的参数
        const bpmInput = document.getElementById('bpmInput');
        const tonguingDelayInput = document.getElementById('tonguingDelayInput');
        
        const bpm = bpmInput.value ? parseFloat(bpmInput.value) : 0;
        const tonguingDelay = parseInt(tonguingDelayInput.value) || 30;
        
        const response = await fetch('/api/playback/start', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                filename: selectedFile.filename,
                instrument: currentInstrument,
                bpm: bpm,
                tonguing_delay: tonguingDelay
            })
        });
        
        const data = await response.json();
        
        if (data.error) {
            showNotification('错误', data.error, 'error');
            startBtn.disabled = false;
            return;
        }
        
        isPlaying = true;
        isPaused = false;
        updateButtonStates();
        showNotification('成功', '演奏已开始', 'success');
        
    } catch (error) {
        console.error('开始演奏失败:', error);
        showNotification('错误', '开始演奏失败，请检查网络连接', 'error');
        startBtn.disabled = false;
    }
}

// 暂停/恢复演奏
async function pausePlayback() {
    if (!isPlaying) return;
    
    try {
        const response = await fetch('/api/playback/pause', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });
        
        const data = await response.json();
        
        if (data.error) {
            showNotification('错误', data.error, 'error');
            return;
        }
        
        isPaused = !isPaused;
        updateButtonStates();
        showNotification('成功', data.message, 'success');
        
    } catch (error) {
        console.error('暂停演奏失败:', error);
        showNotification('错误', '暂停演奏失败，请检查网络连接', 'error');
    }
}

// 停止演奏
async function stopPlayback() {
    if (!isPlaying) return;
    
    try {
        const response = await fetch('/api/playback/stop', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });
        
        const data = await response.json();
        
        if (data.error) {
            showNotification('错误', data.error, 'error');
            return;
        }
        
        isPlaying = false;
        isPaused = false;
        // 不清除selectedFile，这样可以直接重新开始
        updateButtonStates();
        resetStatus();
        showNotification('成功', '演奏已停止', 'success');
        
    } catch (error) {
        console.error('停止演奏失败:', error);
        showNotification('错误', '停止演奏失败，请检查网络连接', 'error');
    }
}

// 更新按钮状态
function updateButtonStates() {
    startBtn.disabled = isPlaying;
    pauseBtn.disabled = !isPlaying;
    stopBtn.disabled = !isPlaying;
    
    if (isPlaying) {
        if (isPaused) {
            pauseBtn.textContent = '▶️ 恢复演奏';
        } else {
            pauseBtn.textContent = '⏸️ 暂停演奏';
        }
    }
}

// 重置状态显示
function resetStatus() {
    currentFileEl.textContent = '-';
    progressEl.textContent = '0%';
    currentNoteEl.textContent = '-';
    totalNotesEl.textContent = '-';
    elapsedTimeEl.textContent = '-';
    playStatusEl.textContent = '未开始';
    progressBarEl.style.width = '0%';
}

// 开始状态更新
function startStatusUpdates() {
    statusUpdateInterval = setInterval(updateStatus, 1000);
}

// 更新状态显示
async function updateStatus() {
	try {
		const response = await fetch('/api/playback/status');
		const status = await response.json();
		
		currentFileEl.textContent = status.current_file || '-';
		progressEl.textContent = `${Math.round(status.progress || 0)}%`;
		currentNoteEl.textContent = status.current_note || '-';
		totalNotesEl.textContent = status.total_notes || '-';
		elapsedTimeEl.textContent = status.elapsed_time || '-';
		
		if (status.is_playing) {
			playStatusEl.textContent = status.is_paused ? '已暂停' : '播放中';
		} else {
			playStatusEl.textContent = '未开始';
		}
		
		progressBarEl.style.width = `${status.progress || 0}%`;
		
		// 检查演奏是否已结束，如果是则重置前端状态
		if (!status.is_playing && isPlaying) {
			isPlaying = false;
			isPaused = false;
			updateButtonStates();
			updateStartButtonState();
		}
		
	} catch (error) {
		console.error('更新状态失败:', error);
	}
}

// 开始日志更新
function startLogUpdates() {
    logUpdateInterval = setInterval(updateLogs, 500);
}

// 更新日志显示
async function updateLogs() {
    try {
        const response = await fetch('/api/playback/logs');
        const data = await response.json();
        
        renderLogs(data.logs);
        
    } catch (error) {
        console.error('更新日志失败:', error);
    }
}

// 渲染日志
function renderLogs(logs) {
    if (!logs || logs.length === 0) {
        logContent.innerHTML = '<div class="no-logs">暂无日志</div>';
        return;
    }
    
    logContent.innerHTML = '';
    
    logs.forEach(log => {
        const logEntry = document.createElement('div');
        logEntry.className = 'log-entry';
        
        const typeClass = log.type === 'info' ? 'info' : 
                         log.type === 'can' ? 'can' : 
                         log.type === 'error' ? 'error' : 'info';
        
        logEntry.innerHTML = `
            <span class="log-timestamp">[${log.timestamp}]</span>
            <span class="log-type ${typeClass}">${log.type.toUpperCase()}</span>
            <span class="log-message">${log.message}</span>
        `;
        
        logContent.appendChild(logEntry);
    });
    
    if (autoScroll) {
        logContent.scrollTop = logContent.scrollHeight;
    }
}

// 清空日志
function clearLogs() {
    logContent.innerHTML = '<div class="no-logs">日志已清空</div>';
}

// 切换自动滚动
function toggleAutoScroll() {
    autoScroll = !autoScroll;
    autoScrollBtn.classList.toggle('active', autoScroll);
    
    if (autoScroll) {
        logContent.scrollTop = logContent.scrollHeight;
    }
}

// 显示通知
function showNotification(title, message, type = 'info') {
    // 创建通知元素
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.innerHTML = `
        <div class="notification-title">${title}</div>
        <div class="notification-message">${message}</div>
    `;
    
    // 添加到页面
    document.body.appendChild(notification);
    
    // 显示动画
    setTimeout(() => {
        notification.classList.add('show');
    }, 100);
    
    // 自动移除
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => {
            document.body.removeChild(notification);
        }, 300);
    }, 3000);
}

// 添加通知样式
const style = document.createElement('style');
style.textContent = `
    .notification {
        position: fixed;
        top: 20px;
        right: 20px;
        background: white;
        border-radius: 8px;
        padding: 15px 20px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        transform: translateX(100%);
        transition: transform 0.3s ease;
        z-index: 1000;
        max-width: 300px;
    }
    
    .notification.show {
        transform: translateX(0);
    }
    
    .notification-title {
        font-weight: bold;
        margin-bottom: 5px;
        color: #2d3748;
    }
    
    .notification-message {
        color: #718096;
        font-size: 14px;
    }
    
    .notification-success {
        border-left: 4px solid #48bb78;
    }
    
    .notification-error {
        border-left: 4px solid #f56565;
    }
    
    .notification-info {
        border-left: 4px solid #3182ce;
    }
    
    .no-files, .no-logs {
        text-align: center;
        padding: 40px;
        color: #718096;
        font-style: italic;
    }
    
    .error {
        text-align: center;
        padding: 20px;
        color: #e53e3e;
        background: #fed7d7;
        border-radius: 8px;
        margin: 10px 0;
    }
`;
document.head.appendChild(style);

////////////////////////////////////////////////////////////////////////////////
// 手动指法测试功能
////////////////////////////////////////////////////////////////////////////////

// 加载指法配置
async function loadFingerings() {
    try {
        loadFingeringsBtn.disabled = true;
        loadFingeringsBtn.textContent = '🔄 加载中...';
        
        const response = await fetch(`/api/fingerings?instrument=${currentInstrument}`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        renderFingeringButtons(data.fingerings);
        
        showNotification('成功', '指法配置加载完成', 'success');
    } catch (error) {
        console.error('加载指法失败:', error);
        showNotification('错误', `加载指法失败: ${error.message}`, 'error');
        fingeringButtonsEl.innerHTML = '<div class="error-message">❌ 加载失败，请重试</div>';
    } finally {
        loadFingeringsBtn.disabled = false;
        loadFingeringsBtn.textContent = '🔄 加载指法';
    }
}

// 渲染指法按钮
function renderFingeringButtons(fingerings) {
    if (!fingerings || fingerings.length === 0) {
        fingeringButtonsEl.innerHTML = '<div class="no-fingerings">📝 暂无指法配置</div>';
        return;
    }
    
    // 按音符名称排序
    fingerings.sort((a, b) => {
        // 简单的音符排序：先按字母，再按数字
        const noteA = a.note.replace(/[#b]/, '');
        const noteB = b.note.replace(/[#b]/, '');
        return noteA.localeCompare(noteB);
    });
    
    fingeringButtonsEl.innerHTML = '';
    
    fingerings.forEach(fingering => {
        const button = document.createElement('button');
        button.className = 'fingering-btn';
        button.textContent = fingering.note;
        
        // 为空指法添加特殊样式
        const hasLeftFingering = fingering.left && fingering.left.length > 0;
        const hasRightFingering = fingering.right && fingering.right.length > 0;
        if (!hasLeftFingering && !hasRightFingering) {
            button.classList.add('empty-fingering');
            button.title = `${fingering.note} - 空指法（左手: 无, 右手: 无）`;
        } else {
            const leftDesc = hasLeftFingering ? fingering.left.join(', ') : '无';
            const rightDesc = hasRightFingering ? fingering.right.join(', ') : '无';
            button.title = `${fingering.note} - 左手: ${leftDesc}, 右手: ${rightDesc}`;
        }
        
        button.addEventListener('click', () => {
            sendFingering(fingering.note);
        });
        
        fingeringButtonsEl.appendChild(button);
    });
}

// 发送单个指法
async function sendFingering(note) {
    try {
        const response = await fetch('/api/fingerings/send', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ 
                note: note,
                instrument: currentInstrument
            })
        });
        
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || `HTTP ${response.status}`);
        }
        
        const data = await response.json();
        showNotification('成功', `已发送音符 ${note} 的指法`, 'success');
        
        // 高亮点击的按钮
        const clickedBtn = Array.from(fingeringButtonsEl.children)
            .find(btn => btn.textContent === note);
        if (clickedBtn) {
            clickedBtn.style.background = '#48bb78';
            clickedBtn.style.color = 'white';
            setTimeout(() => {
                clickedBtn.style.background = '';
                clickedBtn.style.color = '';
            }, 300);
        }
        
    } catch (error) {
        console.error('发送指法失败:', error);
        showNotification('错误', `发送指法失败: ${error.message}`, 'error');
    }
}

// 乐器切换函数
function switchInstrument(instrument) {
    if (isPlaying) {
        showNotification('提示', '演奏进行中，无法切换乐器', 'warning');
        return;
    }
    
    currentInstrument = instrument;
    
    // 更新按钮状态
    sksBtn.classList.toggle('active', instrument === 'sks');
    snBtn.classList.toggle('active', instrument === 'sn');
    
    // 重新加载指法
    loadFingerings();
    
    // 显示切换成功通知
    const instrumentName = instrument === 'sks' ? '萨克斯' : '唢呐（葫芦丝笛子）';
    showNotification('成功', `已切换到${instrumentName}模式`, 'success');
}

// 页面卸载时清理并停止演奏
window.addEventListener('beforeunload', function(e) {
    // 清理定时器
    if (statusUpdateInterval) {
        clearInterval(statusUpdateInterval);
    }
    if (logUpdateInterval) {
        clearInterval(logUpdateInterval);
    }
    
    // 如果正在演奏，发送停止请求（使用同步请求确保执行）
    if (isPlaying) {
        try {
            // 使用sendBeacon确保请求能发出去
            const data = JSON.stringify({});
            navigator.sendBeacon('/api/playback/stop', data);
        } catch (error) {
            console.error('发送停止信号失败:', error);
        }
    }
});

// 监听页面可见性变化（当页面标签页切换时）
document.addEventListener('visibilitychange', function() {
    if (document.hidden && isPlaying) {
        console.log('页面隐藏，演奏将继续进行');
    }
});

// 监听页面失去焦点（可选：用于更严格的控制）
window.addEventListener('pagehide', function() {
    if (isPlaying) {
        try {
            navigator.sendBeacon('/api/playback/stop', JSON.stringify({}));
        } catch (error) {
            console.error('发送停止信号失败:', error);
        }
    }
});

// ========== 歌曲时间轴和空拍编辑功能 ==========

// 加载歌曲时间轴
async function loadSongTimeline(filename) {
    try {
        const response = await fetch(`/api/timeline?filename=${encodeURIComponent(filename)}`);
        const data = await response.json();
        
        if (data.error) {
            console.error('加载时间轴失败:', data.error);
            return;
        }
        
        currentTimeline = data;
        
        // 更新歌曲信息显示
        updateSongInfo();
        
        // 渲染时间轴可视化
        renderTimeline();
        
    } catch (error) {
        console.error('加载时间轴失败:', error);
    }
}

// 更新歌曲信息
function updateSongInfo() {
    if (!currentTimeline) return;
    
    const bpmInput = document.getElementById('bpmInput');
    const bpm = bpmInput.value ? parseFloat(bpmInput.value) : (currentTimeline.bpm || 60);
    
    // 计算总时长
    let totalBeats = 0;
    let restCount = 0;
    let totalNotes = currentTimeline.timeline.length;
    
    currentTimeline.timeline.forEach(item => {
        const duration = parseFloat(item[1]);
        totalBeats += duration;
        if (item[0] === 'NO') {
            restCount++;
        }
    });
    
    const totalSeconds = (totalBeats / bpm) * 60;
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = Math.floor(totalSeconds % 60);
    
    // 更新显示
    document.getElementById('songDuration').textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;
    document.getElementById('songTotalNotes').textContent = totalNotes;
    document.getElementById('songRestCount').textContent = restCount;
}

// 渲染时间轴可视化
function renderTimeline() {
    if (!currentTimeline) return;
    
    const timelineCanvas = document.getElementById('timelineCanvas');
    const bpmInput = document.getElementById('bpmInput');
    const bpm = bpmInput.value ? parseFloat(bpmInput.value) : (currentTimeline.bpm || 60);
    
    // 计算总拍数
    let totalBeats = 0;
    currentTimeline.timeline.forEach(item => {
        totalBeats += parseFloat(item[1]);
    });
    
    // 创建时间轴HTML
    let html = '<div class="timeline-bar">';
    
    currentTimeline.timeline.forEach((item, index) => {
        const note = item[0];
        const duration = parseFloat(item[1]);
        const widthPercent = (duration / totalBeats) * 100;
        const isRest = note === 'NO';
        
        const segmentClass = isRest ? 'rest' : 'note';
        const label = isRest ? 'NO' : note;
        const onclick = isRest ? `openRestEditModal(${index})` : '';
        
        html += `<div class="timeline-segment ${segmentClass}" 
                     style="width: ${widthPercent}%" 
                     onclick="${onclick}"
                     title="${label} (${duration}拍)">
                    ${widthPercent > 3 ? label : ''}
                </div>`;
    });
    
    html += '</div>';
    
    // 添加时间标签
    const totalSeconds = (totalBeats / bpm) * 60;
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = Math.floor(totalSeconds % 60);
    
    html += `<div class="timeline-labels">
                <span>0:00</span>
                <span>${minutes}:${seconds.toString().padStart(2, '0')}</span>
             </div>`;
    
    // 添加图例
    html += `<div class="timeline-legend">
                <div class="legend-item">
                    <div class="legend-color note"></div>
                    <span>正常演奏</span>
                </div>
                <div class="legend-item">
                    <div class="legend-color rest"></div>
                    <span>空拍（可点击编辑）</span>
                </div>
             </div>`;
    
    timelineCanvas.innerHTML = html;
}

// 打开空拍编辑模态框
function openRestEditModal(index) {
    if (!currentTimeline || isPlaying) return;
    
    editingRestIndex = index;
    const item = currentTimeline.timeline[index];
    const duration = parseFloat(item[1]);
    
    document.getElementById('restPosition').textContent = `第 ${index + 1} 个音符`;
    document.getElementById('restCurrentDuration').textContent = duration;
    document.getElementById('restNewDuration').value = duration;
    
    const modal = document.getElementById('restEditModal');
    modal.classList.add('show');
}

// 关闭模态框
function closeRestEditModal() {
    const modal = document.getElementById('restEditModal');
    modal.classList.remove('show');
    editingRestIndex = -1;
}

// 保存空拍修改
async function saveRestEdit() {
    if (editingRestIndex < 0 || !currentTimeline) return;
    
    const newDuration = parseFloat(document.getElementById('restNewDuration').value);
    
    if (newDuration <= 0) {
        showNotification('错误', '时长必须大于0', 'error');
        return;
    }
    
    // 更新本地数据
    currentTimeline.timeline[editingRestIndex][1] = newDuration;
    
    try {
        // 保存到服务器（更新JSON文件）
        const response = await fetch('/api/timeline/update', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                filename: currentTimeline.filename,
                timeline: currentTimeline.timeline
            })
        });
        
        const data = await response.json();
        
        if (data.error) {
            showNotification('错误', `保存失败: ${data.error}`, 'error');
            return;
        }
        
        // 重新渲染
        updateSongInfo();
        renderTimeline();
        
        closeRestEditModal();
        
        showNotification('成功', `已保存空拍时长为 ${newDuration} 拍`, 'success');
        
    } catch (error) {
        console.error('保存失败:', error);
        showNotification('错误', '保存失败，请检查网络连接', 'error');
    }
}

// 初始化BPM输入监听
function initBpmListener() {
    const bpmInput = document.getElementById('bpmInput');
    if (bpmInput) {
        bpmInput.addEventListener('input', function() {
            if (currentTimeline) {
                updateSongInfo();
                renderTimeline();
            }
        });
    }
}

// 初始化模态框事件监听
function initModalListeners() {
    const modal = document.getElementById('restEditModal');
    if (!modal) return;
    
    const closeBtn = modal.querySelector('.modal-close');
    const cancelBtn = document.getElementById('restCancelBtn');
    const saveBtn = document.getElementById('restSaveBtn');
    
    if (closeBtn) closeBtn.addEventListener('click', closeRestEditModal);
    if (cancelBtn) cancelBtn.addEventListener('click', closeRestEditModal);
    if (saveBtn) saveBtn.addEventListener('click', saveRestEdit);
    
    // 点击模态框外部关闭
    modal.addEventListener('click', function(e) {
        if (e.target === modal) {
            closeRestEditModal();
        }
    });
}

// 将函数暴露到全局作用域
window.openRestEditModal = openRestEditModal;
