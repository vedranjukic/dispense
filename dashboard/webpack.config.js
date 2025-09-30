const { composePlugins, withNx } = require('@nx/webpack');
const HtmlWebpackPlugin = require('html-webpack-plugin');

module.exports = composePlugins(
  withNx(),
  (config) => {
    // Dashboard served at root path
    if (!config.output) config.output = {};
    config.output.publicPath = '/';

    // Initialize module rules if not present
    if (!config.module) config.module = {};
    if (!config.module.rules) config.module.rules = [];

    // Add React-specific rules
    config.module.rules.push({
      test: /\.(js|jsx|ts|tsx)$/,
      use: [
        {
          loader: 'babel-loader',
          options: {
            presets: [
              '@babel/preset-env',
              ['@babel/preset-react', { runtime: 'automatic' }],
              '@babel/preset-typescript'
            ]
          }
        }
      ],
      exclude: /node_modules/
    });

    // Add CSS rules
    config.module.rules.push({
      test: /\.css$/,
      use: ['style-loader', 'css-loader']
    });

    // Add HTML plugin
    if (!config.plugins) config.plugins = [];
    config.plugins.push(
      new HtmlWebpackPlugin({
        template: './src/index.html',
        filename: 'index.html',
        inject: true,
        base: '/'
      })
    );

    return config;
  }
);