import { history } from '@src/index'
export default {
  navigate(url: string) {
    history.push(url)
  },
}
