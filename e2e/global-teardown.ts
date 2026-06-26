import fs from 'fs'
import { DB_PATH } from './global-setup'

export default function globalTeardown() {
  if (fs.existsSync(DB_PATH)) {
    fs.unlinkSync(DB_PATH)
  }
}
