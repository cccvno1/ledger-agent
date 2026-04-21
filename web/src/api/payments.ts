import client from './client'

export interface Payment {
  id: string
  customer_id: string
  amount: number
  payment_date: string
  notes: string
  created_at: string
}

export interface CreatePaymentRequest {
  customer_id: string
  amount: number
  payment_date: string
  notes?: string
}

export const paymentApi = {
  list(customerId?: string): Promise<Payment[]> {
    const params = customerId ? { customer_id: customerId } : {}
    return client.get('/payments', { params }).then((r) => r.data)
  },

  create(data: CreatePaymentRequest): Promise<Payment> {
    return client.post('/payments', data).then((r) => r.data)
  },
}
