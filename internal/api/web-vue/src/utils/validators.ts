// Validators - Form Validation Utilities

/**
 * Validate required field
 */
export function required(value: any): true | string {
  if (Array.isArray(value)) {
    return value.length > 0 ? true : '此字段为必填项'
  }
  if (typeof value === 'string') {
    return value.trim().length > 0 ? true : '此字段为必填项'
  }
  return value != null ? true : '此字段为必填项'
}

/**
 * Validate minimum length
 */
export function minLength(min: number) {
  return (value: string): true | string => {
    if (!value) return true // Let required handle empty values
    return value.length >= min ? true : `最少需要 ${min} 个字符`
  }
}

/**
 * Validate maximum length
 */
export function maxLength(max: number) {
  return (value: string): true | string => {
    if (!value) return true
    return value.length <= max ? true : `最多允许 ${max} 个字符`
  }
}

/**
 * Validate email format
 */
export function email(value: string): true | string {
  if (!value) return true
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(value) ? true : '请输入有效的邮箱地址'
}

/**
 * Validate URL format
 */
export function url(value: string): true | string {
  if (!value) return true
  try {
    new URL(value)
    return true
  } catch {
    return '请输入有效的URL地址'
  }
}

/**
 * Validate Twitter username format
 */
export function twitterUsername(value: string): true | string {
  if (!value) return true
  const usernameRegex = /^[a-zA-Z0-9_]{1,15}$/
  return usernameRegex.test(value) ? true : '请输入有效的Twitter用户名（1-15个字符，仅字母数字下划线）'
}

/**
 * Validate cron expression format
 */
export function cronExpression(value: string): true | string {
  if (!value) return true
  const cronRegex = /^(\*|([0-5]?\d)(,([0-5]?\d))*|(\/[0-5]?\d)|([0-5]?\d-[0-5]?\d))( (\*|([0-5]?\d)(,([0-5]?\d))*|(\/[0-5]?\d)|([0-5]?\d-[0-5]?\d))){4}$/
  return cronRegex.test(value) ? true : '请输入有效的Cron表达式'
}

/**
 * Validate number range
 */
export function between(min: number, max: number) {
  return (value: number): true | string => {
    if (value == null) return true
    return value >= min && value <= max ? true : `值必须在 ${min} 到 ${max} 之间`
  }
}

/**
 * Validate positive number
 */
export function positiveNumber(value: number): true | string {
  if (value == null) return true
  return value > 0 ? true : '必须大于0'
}

/**
 * Validate integer
 */
export function integer(value: number): true | string {
  if (value == null) return true
  return Number.isInteger(value) ? true : '必须是整数'
}

/**
 * Run multiple validators and return first error
 */
export function validate(value: any, rules: ((value: any) => true | string)[]): true | string {
  for (const rule of rules) {
    const result = rule(value)
    if (result !== true) return result
  }
  return true
}

/**
 * Validate form object
 */
export function validateForm(formData: Record<string, any>, validationRules: Record<string, ((value: any) => true | string)[]>): { valid: boolean; errors: Record<string, string> } {
  const errors: Record<string, string> = {}
  let valid = true

  for (const [field, rules] of Object.entries(validationRules)) {
    const result = validate(formData[field], rules)
    if (result !== true) {
      errors[field] = result
      valid = false
    }
  }

  return { valid, errors }
}
