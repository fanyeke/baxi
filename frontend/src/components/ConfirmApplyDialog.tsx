import * as Dialog from "@radix-ui/react-dialog"

interface ConfirmApplyDialogProps {
  open: boolean
  onConfirm: () => void
  onCancel: () => void
  title: string
  description: string
  confirmLabel?: string
  destructive?: boolean
}

export function ConfirmApplyDialog({
  open, onConfirm, onCancel, title, description, confirmLabel = "确认执行", destructive,
}: ConfirmApplyDialogProps) {
  return (
    <Dialog.Root open={open} onOpenChange={(v) => { if (!v) onCancel() }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/40 z-40" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-background border rounded-lg shadow-lg p-6 z-50 w-96 max-w-[90vw]">
          <Dialog.Title className="font-semibold text-destructive">{title}</Dialog.Title>
          <Dialog.Description className="text-xs text-muted-foreground mt-2">{description}</Dialog.Description>
          <div className="flex gap-2 mt-4 justify-end">
            <Dialog.Close asChild>
              <button onClick={onCancel} className="px-3 py-1 border rounded text-xs">
                取消
              </button>
            </Dialog.Close>
            <button
              onClick={onConfirm}
              className={`px-3 py-1 rounded text-xs text-white ${destructive !== false ? "bg-destructive" : "bg-primary"}`}
            >
              {confirmLabel}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
