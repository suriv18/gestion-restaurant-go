// Proxy del servidor de desarrollo: el front llama a /api/... (mismo origen) y Angular lo reenvía
// al backend. Así no hace falta CORS. Cambie el destino con la variable API_TARGET:
//   API_TARGET=http://localhost:8080 npm start
const target = process.env.API_TARGET ?? 'http://localhost:18080';

export default {
  '/api': {
    target,
    secure: false,
    changeOrigin: false,
    logLevel: 'warn',
  },
};
