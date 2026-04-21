import client from './client'

export interface QRCodeResponse {
  qrcode: string
  img_content: string
}

export interface QRStatusResponse {
  status: 'wait' | 'scaned' | 'confirmed' | 'expired'
}

export const wechatApi = {
  generateQR(): Promise<QRCodeResponse> {
    return client.post('/wechat/qrcode').then((r) => r.data)
  },

  checkStatus(qrcode: string): Promise<QRStatusResponse> {
    return client.get('/wechat/qrcode/status', { params: { qrcode } }).then((r) => r.data)
  },
}
