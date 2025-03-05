const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const MiniCssExtractPlugin = require("mini-css-extract-plugin");

module.exports = {
    entry: {
        auth: "./src/js/ui/auth_ui.js",
        chats: "./src/js/ui/chats_ui.js",
    },
    output: {
        filename: '[name].bundle.js',
        path: path.resolve(__dirname, 'dist'),
        clean: true,
    },
    mode: 'development',
    devServer: {
        host: '0.0.0.0',
        // static: './dist',
        static: {
            directory: path.resolve(__dirname, 'dist'),
            publicPath: '/',
        },
        port: 8081,
        allowedHosts: [
            'insecuremessenger',
            'localhost'
        ]
    },
    module: {
        rules: [
            {
                test: /\.js$/,
                exclude: /node_modules/,
                use: {
                    loader: "babel-loader",
                },
            },
            // {
            //     test: /\.proto$/,
            //     use: [
            //         {
            //             loader: 'protobufjs-loader'
            //         }
            //     ],
            // },
            {
                test: /\.css$/,
                use: [MiniCssExtractPlugin.loader, "css-loader"],
            }
        ],
    },
    plugins: [
        new MiniCssExtractPlugin({
            filename: "styles/style.css",
        }),
        new HtmlWebpackPlugin({
            filename: 'index.html',
            template: './src/pages/index.html',
            chunks: ['auth'],
        }),
        new HtmlWebpackPlugin({
            filename: "chats.html",
            template: "./src/pages/chats.html",
            chunks: ['chats']
        })
    ],
};