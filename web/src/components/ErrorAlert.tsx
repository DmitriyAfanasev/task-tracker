import { ApiError } from '../api/types'

export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    const details = err.body.details
      ? Object.entries(err.body.details)
          .map(([k, v]) => `${k}: ${v}`)
          .join('; ')
      : ''
    return details ? `${err.message} (${details})` : err.message
  }
  if (err instanceof Error) return err.message
  return 'Неизвестная ошибка'
}

export function ErrorAlert({ error }: { error: unknown }) {
  if (!error) return null
  return (
    <div className="alert alert-error" role="alert">
      {errorMessage(error)}
    </div>
  )
}
