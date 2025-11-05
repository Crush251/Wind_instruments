// 全局变量
let selectedFile = null;
let isPlaying = false;
let autoScroll = true;
let statusUpdateInterval = null;
let logUpdateInterval = null;
let currentInstrument = 'sn'; // 当前选择的乐器：sks(萨克斯) 或 sn(唢呐)
let currentTimeline = null; // 当前加载的时间轴数据
let editingRestIndex = -1; // 正在编辑的空拍索引

// DOM元素（在DOMContentLoaded后初始化）
let searchInput, searchBtn, fileList, startBtn, stopBtn;
let clearLogBtn, autoScrollBtn, logContent, loadFingeringsBtn, fingeringButtonsEl;
let sksBtn, snBtn;
let currentFileEl, progressEl, currentNoteEl, totalNotesEl;
let elapsedTimeEl, playStatusEl, progressBarEl;

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    // 初始化DOM元素引用
    searchInput = document.getElementById('searchInput');
    searchBtn = document.getElementById('searchBtn');
    fileList = document.getElementById('fileList');
    startBtn = document.getElementById('startBtn');
    stopBtn = document.getElementById('stopBtn');
    clearLogBtn = document.getElementById('clearLogBtn');
    autoScrollBtn = document.getElementById('autoScrollBtn');
    logContent = document.getElementById('logContent');
    loadFingeringsBtn = document.getElementById('loadFingeringsBtn');
    fingeringButtonsEl = document.getElementById('fingeringButtons');
    sksBtn = document.getElementById('sksBtn');
    snBtn = document.getElementById('snBtn');
    currentFileEl = document.getElementById('currentFile');
    progressEl = document.getElementById('progress');
    currentNoteEl = document.getElementById('currentNote');
    totalNotesEl = document.getElementById('totalNotes');
    elapsedTimeEl = document.getElementById('elapsedTime');
    playStatusEl = document.getElementById('playStatus');
    progressBarEl = document.getElementById('progressBar');
    
    loadMusicFiles();
    setupEventListeners();
    startStatusUpdates();
    startLogUpdates();
    loadFingerings(); // 自动加载指法
    
    // 初始化模态框事件监听
    initModalListeners();
    
    // 初始化BPM输入监听
    initBpmListener();
    
    // 初始化预处理按钮
    initPreprocessButton();
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
    
    // 气泵调试按钮
    const pumpDebugBtn = document.getElementById('pumpDebugBtn');
    const pumpOnBtn = document.getElementById('pumpOnBtn');
    const pumpOffBtn = document.getElementById('pumpOffBtn');
    const pumpDebugInput = document.getElementById('pumpDebugInput');
    if (pumpDebugBtn && pumpDebugInput) {
        pumpDebugBtn.addEventListener('click', sendPumpDebugCommand);
        pumpDebugInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendPumpDebugCommand();
            }
        });
    }
    if (pumpOnBtn && pumpOffBtn) {
        pumpOnBtn.addEventListener('click', function() {
            sendPumponAndOff('on');
        });
        pumpOffBtn.addEventListener('click', function() {
            sendPumponAndOff('off');
        });
    }
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
    
    // 检查执行序列缓存
    checkExecCache();
}

// 更新开始按钮状态
function updateStartButtonState() {
    startBtn.disabled = !selectedFile || isPlaying;
}

// 开始演奏（强制使用预计算模式）
async function startPlayback() {
    if (!selectedFile || isPlaying) return;
    
    try {
        startBtn.disabled = true;
        
        // 重置计时器显示（开始新播放时）
        stopTimer();
        updateTimerDisplay(0);
        document.getElementById('timeDiff').textContent = '-';
        
        // 隐藏之前的空拍详情
        hideSignificantRests();
        
        // 获取用户输入的参数
        const bpmInput = document.getElementById('bpmInput');
        const tonguingDelayInput = document.getElementById('tonguingDelayInput');
        
        const bpm = bpmInput.value ? parseFloat(bpmInput.value) : 0;
        const tonguingDelay = parseInt(tonguingDelayInput.value) || 30;
        
        // 检查是否已有预计算文件
        if (!currentExecFile) {
            // 没有预计算文件，自动进行预处理
            showNotification('提示', '正在自动预处理...', 'info');
            updatePreprocessStatus('🔄 自动预处理中...', 'loading');
            
            const preprocessSuccess = await preprocessAndWait(bpm, tonguingDelay);
            if (!preprocessSuccess) {
                startBtn.disabled = false;
                return;
            }
        }
        
        // 使用预计算执行序列播放
        const success = await playExecSequence();
        if (success) {
            isPlaying = true;
            updateButtonStates();
            startTimer();
        } else {
            startBtn.disabled = false;
        }
        
    } catch (error) {
        console.error('开始演奏失败:', error);
        showNotification('错误', '开始演奏失败，请检查网络连接', 'error');
        startBtn.disabled = false;
    }
}

// 预处理并等待完成
async function preprocessAndWait(bpm, tonguingDelay) {
    try {
        const response = await fetch('/api/preprocess', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                source_file: selectedFile.file_path || selectedFile.filename,
                instrument: currentInstrument,
                bpm: bpm,
                tonguing_delay: tonguingDelay
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            currentExecFile = data.exec_file;
            theoreticalDuration = data.duration_sec;
            updatePreprocessStatus(`✅ 自动预处理完成！时长: ${data.duration_sec.toFixed(2)}秒`, 'success');
            updateSongDuration(data.duration_sec);
            return true;
        } else {
            updatePreprocessStatus(`❌ 预处理失败: ${data.error}`, 'error');
            showNotification('错误', `预处理失败: ${data.error}`, 'error');
            return false;
        }
    } catch (error) {
        console.error('预处理失败:', error);
        updatePreprocessStatus('❌ 预处理失败: 网络错误', 'error');
        showNotification('错误', '预处理失败: 网络错误', 'error');
        return false;
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
        // 不清除selectedFile，这样可以直接重新开始
        updateButtonStates();
        // 不调用 resetStatus()，保留最终计时结果显示
        stopTimer(); // 停止计时器
        showNotification('成功', '演奏已停止', 'success');
        
    } catch (error) {
        console.error('停止演奏失败:', error);
        showNotification('错误', '停止演奏失败，请检查网络连接', 'error');
    }
}

// 更新按钮状态
function updateButtonStates() {
    startBtn.disabled = isPlaying;
    stopBtn.disabled = !isPlaying;
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
			playStatusEl.textContent = '播放中';
		} else {
			playStatusEl.textContent = '未开始';
		}
		
		progressBarEl.style.width = `${status.progress || 0}%`;
		
		// 检查演奏是否已结束，如果是则重置前端状态并显示空拍信息
		if (!status.is_playing && isPlaying) {
			isPlaying = false;
			updateButtonStates();
			updateStartButtonState();
			pauseTimerAtEnd(); // 暂停计时器但保留最终显示
			
			// 显示播放结束后的统计信息（包括空拍）
			console.log('播放结束，检查空拍数据:', status.significant_rests);
			if (status.significant_rests && status.significant_rests.length > 0) {
				console.log('显示', status.significant_rests.length, '个显著空拍');
				displaySignificantRests(status.significant_rests);
			} else {
				console.log('没有显著空拍数据');
			}
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
        
        // 自动填充BPM输入框
        const bpmInput = document.getElementById('bpmInput');
        if (bpmInput && data.bpm) {
            bpmInput.value = data.bpm;
        }
        
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

////////////////////////////////////////////////////////////////////////////////
// 预处理和执行序列相关功能
////////////////////////////////////////////////////////////////////////////////

let currentExecFile = null;
let theoreticalDuration = 0;
let timerInterval = null;
let timerStartTime = null;
let pausedTime = 0;

// 初始化预处理按钮
function initPreprocessButton() {
    const preprocessBtn = document.getElementById('preprocessBtn');
    const useCacheCheckbox = document.getElementById('useCacheCheckbox');
    
    if (preprocessBtn) {
        preprocessBtn.addEventListener('click', handlePreprocess);
    }
    
    // 文件选择或参数变化时检查缓存
    document.getElementById('bpmInput')?.addEventListener('change', checkExecCache);
    document.getElementById('tonguingDelayInput')?.addEventListener('change', checkExecCache);
}

// 检查执行序列缓存
async function checkExecCache() {
    if (!selectedFile) return;
    
    const bpm = document.getElementById('bpmInput').value || '0';
    const tonguingDelay = document.getElementById('tonguingDelayInput').value || '30';
    const instrument = currentInstrument;
    
    try {
        const sourceFile = selectedFile.file_path || selectedFile.filename;
        const response = await fetch(`/api/exec/check?source_file=${encodeURIComponent(sourceFile)}&instrument=${instrument}&bpm=${bpm}&tonguing_delay=${tonguingDelay}`);
        const data = await response.json();
        
        if (data.exists) {
            currentExecFile = data.exec_file;
            theoreticalDuration = data.duration_sec;
            updatePreprocessStatus(`✅ 找到缓存文件（时长: ${data.duration_sec.toFixed(2)}秒）`, 'success');
            updateSongDuration(data.duration_sec);
        } else {
            currentExecFile = null;
            updatePreprocessStatus('ℹ️ 未找到缓存，点击开始将自动生成', 'info');
        }
    } catch (error) {
        console.error('检查缓存失败:', error);
        updatePreprocessStatus('❌ 检查缓存失败', 'error');
    }
}

// 处理预处理请求
async function handlePreprocess() {
    if (!selectedFile) {
        updatePreprocessStatus('❌ 请先选择音乐文件', 'error');
        return;
    }
    
    const bpm = parseFloat(document.getElementById('bpmInput').value) || 0;
    const tonguingDelay = parseInt(document.getElementById('tonguingDelayInput').value) || 30;
    const instrument = currentInstrument;
    
    updatePreprocessStatus('🔄 正在预处理...', 'loading');
    
    const preprocessBtn = document.getElementById('preprocessBtn');
    if (preprocessBtn) preprocessBtn.disabled = true;
    
    try {
        const response = await fetch('/api/preprocess', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                source_file: selectedFile.file_path || selectedFile.filename,
                instrument: instrument,
                bpm: bpm,
                tonguing_delay: tonguingDelay
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            currentExecFile = data.exec_file;
            theoreticalDuration = data.duration_sec;
            updatePreprocessStatus(`✅ 预处理完成！时长: ${data.duration_sec.toFixed(2)}秒，事件数: ${data.total_events}`, 'success');
            updateSongDuration(data.duration_sec);
        } else {
            updatePreprocessStatus(`❌ 预处理失败: ${data.error}`, 'error');
        }
    } catch (error) {
        console.error('预处理失败:', error);
        updatePreprocessStatus('❌ 预处理失败: 网络错误', 'error');
    } finally {
        if (preprocessBtn) preprocessBtn.disabled = false;
    }
}

// 更新预处理状态显示
function updatePreprocessStatus(message, type) {
    const statusElement = document.getElementById('preprocessStatus');
    if (!statusElement) return;
    
    statusElement.textContent = message;
    statusElement.className = `preprocess-status ${type}`;
}

// 更新歌曲时长显示
function updateSongDuration(durationSec) {
    const durationElement = document.getElementById('songDuration');
    if (durationElement) {
        const minutes = Math.floor(durationSec / 60);
        const seconds = (durationSec % 60).toFixed(2);
        durationElement.textContent = `${minutes}:${seconds.padStart(5, '0')}`;
    }
}

// 启动计时器
function startTimer() {
    timerStartTime = Date.now() - pausedTime;
    pausedTime = 0;
    
    timerInterval = setInterval(() => {
        const elapsed = (Date.now() - timerStartTime) / 1000;
        updateTimerDisplay(elapsed);
        
        // 计算时间误差
        if (theoreticalDuration > 0) {
            const diff = elapsed - theoreticalDuration;
            const diffPercent = (diff / theoreticalDuration * 100).toFixed(2);
            document.getElementById('timeDiff').textContent = `${diff >= 0 ? '+' : ''}${diff.toFixed(3)}s (${diffPercent}%)`;
        }
    }, 10); // 每10ms更新一次，显示毫秒
}

// 暂停计时器
function pauseTimer() {
    if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
        pausedTime = Date.now() - timerStartTime;
    }
}

// 停止计时器（用于手动停止或开始新播放）
function stopTimer() {
    if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
    }
    timerStartTime = null;
    pausedTime = 0;
    updateTimerDisplay(0);
    document.getElementById('timeDiff').textContent = '-';
}

// 暂停计时器但保留显示（用于播放自然结束）
function pauseTimerAtEnd() {
    if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
    }
    // 保留 timerStartTime 和 pausedTime，不清零显示
    // 这样最终的时间和误差会保留在界面上
}

// 更新计时器显示
function updateTimerDisplay(seconds) {
    const timerElement = document.getElementById('actualTimer');
    if (!timerElement) return;
    
    const minutes = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    const ms = Math.floor((seconds % 1) * 1000);
    
    timerElement.textContent = `${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}.${ms.toString().padStart(3, '0')}`;
}

// 播放执行序列
async function playExecSequence() {
    if (!currentExecFile) {
        showNotification('错误', '请先预处理或选择缓存文件', 'error');
        return false;
    }
    
    try {
        const response = await fetch('/api/exec/play', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                exec_file: currentExecFile
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            showNotification('成功', '开始播放执行序列', 'success');
            return true;
        } else {
            showNotification('错误', `播放失败: ${data.error}`, 'error');
            return false;
        }
    } catch (error) {
        console.error('播放失败:', error);
        showNotification('错误', '播放失败: 网络错误', 'error');
        return false;
    }
}

////////////////////////////////////////////////////////////////////////////////
// 显著空拍显示功能
////////////////////////////////////////////////////////////////////////////////

// 显示显著空拍详情
function displaySignificantRests(rests) {
    console.log('displaySignificantRests 被调用，数据:', rests);
    
    const restDetailsSection = document.getElementById('restDetailsSection');
    const restDetailsContent = document.getElementById('restDetailsContent');
    const significantRestCount = document.getElementById('significantRestCount');
    
    console.log('DOM元素:', { restDetailsSection, restDetailsContent, significantRestCount });
    
    if (!restDetailsSection || !restDetailsContent) {
        console.error('DOM元素未找到！');
        return;
    }
    
    // 更新显著空拍数量
    if (significantRestCount) {
        significantRestCount.textContent = rests.length;
    }
    
    if (rests.length === 0) {
        restDetailsSection.style.display = 'none';
        return;
    }
    
    // 显示区域
    restDetailsSection.style.display = 'block';
    console.log('显示区域已展开');
    
    // 清空现有内容
    restDetailsContent.innerHTML = '';
    
    // 生成每个空拍的详情
    rests.forEach((rest, index) => {
        console.log(`生成空拍${index + 1}:`, rest);
        console.log(`  start_offset: ${rest.start_offset}, 格式化: ${formatTime(rest.start_offset)}`);
        console.log(`  end_offset: ${rest.end_offset}, 格式化: ${formatTime(rest.end_offset)}`);
        
        const restItem = document.createElement('div');
        restItem.className = 'rest-item';
        
        restItem.innerHTML = `
            <div class="rest-label">空拍${index + 1}</div>
            <div class="rest-time">
                <span class="label">起始时间</span>
                <span class="value">${formatTime(rest.start_offset)}</span>
            </div>
            <div class="rest-time">
                <span class="label">结束时间</span>
                <span class="value">${formatTime(rest.end_offset)}</span>
            </div>
            <div class="rest-duration">
                持续: ${rest.duration.toFixed(2)}s (${rest.beats.toFixed(1)}拍)
            </div>
        `;
        
        restDetailsContent.appendChild(restItem);
        console.log(`空拍${index + 1} DOM已添加`);
    });
    
    console.log('所有空拍详情已生成');
}

// 格式化时间显示（秒转为 分:秒.毫秒 格式）
function formatTime(seconds) {
    const minutes = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    const ms = Math.floor((seconds % 1) * 1000);
    return `${minutes}:${secs.toString().padStart(2, '0')}.${ms.toString().padStart(3, '0')}`;
}

// 在开始新播放时隐藏空拍详情
function hideSignificantRests() {
    const restDetailsSection = document.getElementById('restDetailsSection');
    if (restDetailsSection) {
        restDetailsSection.style.display = 'none';
    }
    const significantRestCount = document.getElementById('significantRestCount');
    if (significantRestCount) {
        significantRestCount.textContent = '-';
    }
}
//sendto气泵
async function sendPumponAndOff(command) {
    console.log('sendPumponAndOff 被调用，命令:', command);
    const statusEl = document.getElementById('pumpDebugStatus');
    const response = await fetch('/api/pump/debug', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ command: command })
    });
    const data = await response.json();
    if (response.ok) {
        statusEl.textContent = `✅ ${data.message}`;
        statusEl.className = 'pump-debug-status success';
    } else {
        statusEl.textContent = `❌ ${data.error}${data.details ? ': ' + data.details : ''}`;
        statusEl.className = 'pump-debug-status error';
    }
    setTimeout(() => {
        statusEl.textContent = '';
        statusEl.className = 'pump-debug-status';
    }, 3000);
}

// 发送气泵调试命令
async function sendPumpDebugCommand() {
    const input = document.getElementById('pumpDebugInput');
    const statusEl = document.getElementById('pumpDebugStatus');
    const command = input.value.trim();
    
    if (!command) {
        statusEl.textContent = '⚠️ 请输入命令';
        statusEl.className = 'pump-debug-status warning';
        return;
    }
    
    try {
        statusEl.textContent = '⏳ 发送中...';
        statusEl.className = 'pump-debug-status info';
        
        const response = await fetch('/api/pump/debug', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ command: command })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            statusEl.textContent = `✅ ${data.message}`;
            statusEl.className = 'pump-debug-status success';
            input.value = ''; // 清空输入框
        } else {
            statusEl.textContent = `❌ ${data.error}${data.details ? ': ' + data.details : ''}`;
            statusEl.className = 'pump-debug-status error';
        }
    } catch (error) {
        console.error('发送气泵命令失败:', error);
        statusEl.textContent = `❌ 发送失败: ${error.message}`;
        statusEl.className = 'pump-debug-status error';
    }
    
    // 3秒后清除状态
    setTimeout(() => {
        statusEl.textContent = '';
        statusEl.className = 'pump-debug-status';
    }, 3000);
}
