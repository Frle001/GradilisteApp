export type InvoiceType = 'materijal' | 'gorivo' | 'leasing' | 'alati' | 'oprema'

export type Supplier = 'Energy Centar' | 'Pondeljak' | 'Lipa Promet'

export type LeasingCompany = 'Impuls' | 'Unicredit Leasing'

export const INVOICE_TYPE_LABELS: Record<InvoiceType, string> = {
  materijal: 'Materijal',
  gorivo: 'Gorivo',
  leasing: 'Leasing',
  alati: 'Alati',
  oprema: 'Oprema',
}

export const SUPPLIERS: Supplier[] = ['Energy Centar', 'Pondeljak', 'Lipa Promet']

export const LEASING_COMPANIES: LeasingCompany[] = ['Impuls', 'Unicredit Leasing']

export interface FinanceDocMeta {
  original_filename: string
  mime_type: string
  file_size: number
  uploaded_at: string
}

export interface CompanyInvoice {
  id: string
  invoice_type: InvoiceType
  supplier?: string
  leasing_company?: string
  document: FinanceDocMeta
  created_by: string
  created_by_name: string
  created_at: string
}

export interface R1Receipt {
  id: string
  submitted_by: string
  submitter_name: string
  price: number
  document: FinanceDocMeta
  created_at: string
}
