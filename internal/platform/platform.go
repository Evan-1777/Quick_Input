// Package platform 封装与操作系统交互的原生能力。
package platform

// Paster 把文本键入到唤出面板之前的前台窗口。
type Paster interface {
	// CaptureTarget 记录当前前台窗口，已属于本进程时保留旧快照。
	CaptureTarget()
	// Paste 恢复目标焦点，以剪贴板粘贴文本并尽量还原剪贴板。
	Paste(text string) error
}

// restoreClipboard 判断键入后是否可以恢复备份文本：仅当键入前有文本，
// 且期间剪贴板序列号未被其他程序改变。
func restoreClipboard(hadText bool, typed, current uint32) bool {
	return hadText && typed == current
}
