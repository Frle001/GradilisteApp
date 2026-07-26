'use client'

import { useState } from 'react'
import AssetInventoryList from './AssetInventoryList'
import MaterialResponsibilityList from './MaterialResponsibilityList'
import TransferModal from './TransferModal'
import type {
  InventoryResponse,
  InventoryAssetItem,
  InventoryMaterialItem,
} from '@/lib/types/inventory'

type TransferSubject =
  | { kind: 'asset'; item: InventoryAssetItem }
  | { kind: 'material'; item: InventoryMaterialItem }

interface Props {
  inventory: InventoryResponse
  canInitiateTransfer: boolean
  onTransferSuccess: () => void
}

function SectionCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-4">
      <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">{title}</h2>
      {children}
    </div>
  )
}

export default function InventoryOverview({ inventory, canInitiateTransfer, onTransferSuccess }: Props) {
  const [transferSubject, setTransferSubject] = useState<TransferSubject | null>(null)

  function handleAssetTransfer(asset: InventoryAssetItem) {
    setTransferSubject({ kind: 'asset', item: asset })
  }

  function handleMaterialTransfer(material: InventoryMaterialItem) {
    setTransferSubject({ kind: 'material', item: material })
  }

  return (
    <>
      <div className="space-y-5">
        <SectionCard title="Oprema i sredstva">
          <AssetInventoryList
            assets={inventory.assets}
            canInitiateTransfer={canInitiateTransfer}
            onTransfer={handleAssetTransfer}
          />
        </SectionCard>

        <SectionCard title="Materijali u odgovornosti">
          <MaterialResponsibilityList
            materials={inventory.materials}
            canInitiateTransfer={canInitiateTransfer}
            onTransfer={handleMaterialTransfer}
          />
        </SectionCard>
      </div>

      <TransferModal
        subject={transferSubject}
        onClose={() => setTransferSubject(null)}
        onSuccess={onTransferSuccess}
        onNeedsRefresh={onTransferSuccess}
      />
    </>
  )
}
