import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { VersionInfo } from '@/services/version'
import type { UpdateStatusResponse } from '@/services/api'

/**
 * 系统状态管理 Store
 *
 * 职责：
 * - 管理系统运行状态（running/error/connecting）
 * - 管理版本信息和版本检查状态
 * - 管理 Fuzzy 模式加载状态
 */
export const useSystemStore = defineStore('system', () => {
  // ===== 状态 =====

  // 系统连接状态
  type SystemStatus = 'running' | 'error' | 'connecting'
  const systemStatus = ref<SystemStatus>('connecting')

  // 版本信息
  const versionInfo = ref<VersionInfo>({
    currentVersion: '',
    latestVersion: null,
    isLatest: false,
    hasUpdate: false,
    releaseUrl: null,
    lastCheckTime: 0,
    status: 'checking',
  })

  // 版本检查加载状态
  const isCheckingVersion = ref(false)

  // Fuzzy 模式加载状态
  const fuzzyModeLoading = ref(false)
  const fuzzyModeLoadError = ref(false)

  // 移除计费头加载状态
  const stripBillingHeaderLoading = ref(false)
  const stripBillingHeaderLoadError = ref(false)

  // OTA 更新状态
  const updateStatus = ref<UpdateStatusResponse | null>(null)
  const isUpdating = ref(false)
  const updateDialogOpen = ref(false)

  // ===== 计算属性 =====

  // ===== 操作方法 =====

  /**
   * 设置系统状态
   */
  function setSystemStatus(status: SystemStatus) {
    systemStatus.value = status
  }

  /**
   * 设置版本信息
   */
  function setVersionInfo(info: VersionInfo) {
    versionInfo.value = info
  }

  /**
   * 更新当前版本号
   */
  function setCurrentVersion(version: string) {
    versionInfo.value.currentVersion = version
  }

  /**
   * 设置版本检查状态
   */
  function setCheckingVersion(checking: boolean) {
    isCheckingVersion.value = checking
  }

  /**
   * 设置 Fuzzy 模式加载状态
   */
  function setFuzzyModeLoading(loading: boolean) {
    fuzzyModeLoading.value = loading
  }

  /**
   * 设置 Fuzzy 模式加载错误状态
   */
  function setFuzzyModeLoadError(error: boolean) {
    fuzzyModeLoadError.value = error
  }

  /**
   * 设置移除计费头加载状态
   */
  function setStripBillingHeaderLoading(loading: boolean) {
    stripBillingHeaderLoading.value = loading
  }

  /**
   * 设置移除计费头加载错误状态
   */
  function setStripBillingHeaderLoadError(error: boolean) {
    stripBillingHeaderLoadError.value = error
  }

  function setUpdateStatus(status: UpdateStatusResponse | null) {
    updateStatus.value = status
  }

  function setIsUpdating(updating: boolean) {
    isUpdating.value = updating
  }

  function setUpdateDialogOpen(open: boolean) {
    updateDialogOpen.value = open
  }

  /**
   * 重置系统状态
   */
  function resetSystemState() {
    systemStatus.value = 'connecting'
    versionInfo.value = {
      currentVersion: '',
      latestVersion: null,
      isLatest: false,
      hasUpdate: false,
      releaseUrl: null,
      lastCheckTime: 0,
      status: 'checking',
    }
    isCheckingVersion.value = false
    fuzzyModeLoading.value = false
    fuzzyModeLoadError.value = false
    stripBillingHeaderLoading.value = false
    stripBillingHeaderLoadError.value = false
  }

  return {
    // 状态
    systemStatus,
    versionInfo,
    isCheckingVersion,
    fuzzyModeLoading,
    fuzzyModeLoadError,
    stripBillingHeaderLoading,
    stripBillingHeaderLoadError,
    updateStatus,
    isUpdating,
    updateDialogOpen,

    // 计算属性

    // 方法
    setSystemStatus,
    setVersionInfo,
    setCurrentVersion,
    setCheckingVersion,
    setFuzzyModeLoading,
    setFuzzyModeLoadError,
    setStripBillingHeaderLoading,
    setStripBillingHeaderLoadError,
    setUpdateStatus,
    setIsUpdating,
    setUpdateDialogOpen,
    resetSystemState,
  }
})
