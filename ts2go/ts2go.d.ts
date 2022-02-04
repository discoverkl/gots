
interface Base {
  context: Context
  [propName: string]: any
}

interface Context {
  withCancel(): [any, CancelFunc]
}

interface CancelFunc {
  (): void
}

export function getapi(options?: {
  async?: false
  publicPath?: string
}): Base

export function getapi(options: {
  async: true
  publicPath?: string
}): Promise<Base>