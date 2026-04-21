import client from './client'

export interface Customer {
  id: string
  name: string
  aliases: string[]
  created_at: string
}

export interface CustomerSummary {
  customer_id: string
  customer_name: string
  total_amount: number
  settled_amount: number
  pending_amount: number
  entry_count: number
}

export const customerApi = {
  list(): Promise<Customer[]> {
    return client.get('/customers').then((r) => r.data)
  },

  get(id: string): Promise<Customer> {
    return client.get(`/customers/${id}`).then((r) => r.data)
  },

  create(name: string): Promise<Customer> {
    return client.post('/customers', { name }).then((r) => r.data)
  },

  summary(customerId: string): Promise<CustomerSummary[]> {
    return client.get(`/customers/${customerId}/summary`).then((r) => r.data)
  },

  settle(customerId: string): Promise<void> {
    return client.post(`/customers/${customerId}/settle`).then(() => undefined)
  },
}
