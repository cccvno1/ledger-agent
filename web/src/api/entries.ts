import client from './client'

export interface Entry {
  id: string
  customer_id: string
  customer_name: string
  product_name: string
  unit_price: number
  quantity: number
  unit: string
  amount: number
  entry_date: string
  is_settled: boolean
  settled_at: string | null
  notes: string
  created_at: string
  updated_at: string
}

export interface CreateEntryRequest {
  customer_id: string
  customer_name: string
  product_name: string
  unit_price: number
  quantity: number
  unit?: string
  entry_date: string
  notes?: string
}

export interface UpdateEntryRequest {
  product_name?: string
  unit_price?: number
  quantity?: number
  unit?: string
  entry_date?: string
  notes?: string
}

export interface ListEntriesParams {
  customer_id?: string
  date_from?: string
  date_to?: string
  is_settled?: boolean
}

export const entryApi = {
  list(params: ListEntriesParams = {}): Promise<Entry[]> {
    return client.get('/entries', { params }).then((r) => r.data)
  },

  create(data: CreateEntryRequest): Promise<Entry> {
    return client.post('/entries', data).then((r) => r.data)
  },

  update(id: string, data: UpdateEntryRequest): Promise<Entry> {
    return client.put(`/entries/${id}`, data).then((r) => r.data)
  },

  delete(id: string): Promise<void> {
    return client.delete(`/entries/${id}`).then(() => undefined)
  },
}
