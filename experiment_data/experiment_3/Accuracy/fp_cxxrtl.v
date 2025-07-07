`timescale 1ns/1ps

module a (
    input  wire       in0,
    input  wire       clock_0,
    input  wire       clock_1,
    output wire [5:0] out1
);

    reg [5:0] reg_0;
    reg [5:0] reg_1;

    initial begin
        reg_0 = 0;
        reg_1 = 0;
    end

    assign out1 = (reg_1 - reg_0);

    always @(posedge clock_0 or posedge clock_1) begin
        if (clock_1) begin
            reg_0 <= 1;
            reg_1 <= 2;
        end else begin
            reg_0 <= 1;
            reg_1 <= 3;
        end
    end

endmodule